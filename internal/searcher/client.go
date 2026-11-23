// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/redact"
)

const searchAPIEndpoint = "https://www.nixsearch.com"

var ErrNotFound = errors.New("Not found")

type client struct {
	host string
}

func Client() *client {
	return &client{
		host: envir.GetValueOrDefault(envir.DevboxSearchHost, searchAPIEndpoint),
	}
}

func (c *client) Host() string {
	return c.host
}

// Search returns the packages that match query, most relevant first. A
// positive limit caps both the number of packages and each package's newest
// releases; otherwise every package and release is returned.
func (c *client) Search(ctx context.Context, query string, limit int) (*SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("query should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "v2/search")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	releases := "all"
	if limit > 0 {
		releases = strconv.Itoa(limit)
	}
	searchURL := endpoint + "?q=" + url.QueryEscape(query) + "&releases=" + releases

	searchResults, err := execGet[SearchResultsV2](ctx, searchURL)
	if err != nil {
		return nil, err
	}
	hits := searchResults.Results
	if limit > 0 && len(hits) > limit {
		hits = hits[:limit]
	}

	found := make([]*Package, len(hits))
	var lookups []int
	for i, hit := range hits {
		if hit.Releases == nil {
			lookups = append(lookups, i)
			continue
		}
		versions := make([]SearchRelease, len(hit.Releases))
		for j, release := range hit.Releases {
			versions[j] = SearchRelease{Version: release.Version, Systems: sortedSystems(release.Systems)}
		}
		found[i] = &Package{Name: hit.Name, Versions: versions}
	}
	if err := c.lookupPackages(ctx, hits, lookups, limit, found); err != nil {
		return nil, err
	}

	results := &SearchResults{NumResults: searchResults.TotalResults}
	for _, pkg := range found {
		if pkg != nil {
			results.Packages = append(results.Packages, *pkg)
		}
	}
	return results, nil
}

// lookupPackages fills found[i] from /v2/pkg for each index in lookups, for
// search services that don't return releases inline. A package whose lookup
// fails is left out, unless every lookup fails.
func (c *client) lookupPackages(ctx context.Context, hits []SearchResultV2, lookups []int, limit int, found []*Package) error {
	if len(lookups) == 0 {
		return nil
	}
	pkgEndpoint, err := url.JoinPath(c.host, "v2/pkg")
	if err != nil {
		return errors.WithStack(err)
	}

	errs := make([]error, len(lookups))
	var group errgroup.Group
	group.SetLimit(8)
	for n, i := range lookups {
		group.Go(func() error {
			pkgURL := pkgEndpoint + "?name=" + url.QueryEscape(hits[i].Name)
			packageInfo, err := execGet[PackageV2](ctx, pkgURL)
			errs[n] = err
			if err == nil {
				pkg := newPackage(packageInfo, limit)
				found[i] = &pkg
			}
			return nil
		})
	}
	_ = group.Wait()

	var firstErr error
	for _, err := range errs {
		if err == nil {
			return nil
		}
		if firstErr == nil && !errors.Is(err, ErrNotFound) {
			firstErr = err
		}
	}
	return firstErr
}

func newPackage(info *PackageV2, limit int) Package {
	releases := info.Releases
	if limit > 0 && len(releases) > limit {
		releases = releases[:limit]
	}
	versions := make([]SearchRelease, len(releases))
	for i, release := range releases {
		systems := make([]string, len(release.Platforms))
		for j, platform := range release.Platforms {
			systems[j] = platform.nixSystem()
		}
		versions[i] = SearchRelease{Version: release.Version, Systems: sortedSystems(systems)}
	}
	return Package{Name: info.Name, Versions: versions}
}

func sortedSystems(systems []string) []string {
	sorted := slices.Clone(systems)
	slices.Sort(sorted)
	return slices.Compact(sorted)
}

func (p Platform) nixSystem() string {
	arch := strings.ReplaceAll(p.Arch, "-", "_")
	if arch == "arm64" {
		arch = "aarch64"
	}
	goos := strings.ToLower(p.OS)
	if goos == "macos" {
		goos = "darwin"
	}
	return arch + "-" + goos
}

// Resolve calls the /resolve endpoint of the search service. This returns
// the latest version of the package that matches the version constraint.
func (c *client) Resolve(name, version string) (*PackageVersion, error) {
	if name == "" || version == "" {
		return nil, fmt.Errorf("name and version should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "v1/resolve")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	searchURL := endpoint +
		"?name=" + url.QueryEscape(name) +
		"&version=" + url.QueryEscape(version)

	return execGet[PackageVersion](context.TODO(), searchURL)
}

// Resolve calls the /resolve endpoint of the search service. This returns
// the latest version of the package that matches the version constraint.
func (c *client) ResolveV2(ctx context.Context, name, version string) (*ResolveResponse, error) {
	if name == "" {
		return nil, redact.Errorf("name is empty")
	}
	if version == "" {
		return nil, redact.Errorf("version is empty")
	}

	endpoint, err := url.JoinPath(c.host, "v2/resolve")
	if err != nil {
		return nil, redact.Errorf("invalid search endpoint host %q: %w", redact.Safe(c.host), redact.Safe(err))
	}
	searchURL := endpoint +
		"?name=" + url.QueryEscape(name) +
		"&version=" + url.QueryEscape(version)

	return execGet[ResolveResponse](ctx, searchURL)
}

var userAgent = fmt.Sprintf("Devbox/%s (%s; %s)", build.Version, runtime.GOOS, runtime.GOARCH)

// ErrUnavailable is returned when the search service can't be reached or keeps
// returning server errors after all retries.
var ErrUnavailable = errors.New("search service unavailable")

// maxRetries is the number of times a request is retried after the first
// attempt fails with a network error, a 5xx, or a 429.
const maxRetries = 3

// retryBackoff returns how long to wait before retry number attempt (1-based).
// It's a variable so tests can make it instant.
var retryBackoff = func(attempt int) time.Duration {
	return time.Duration(attempt) * 250 * time.Millisecond
}

func execGet[T any](ctx context.Context, url string) (*T, error) {
	var lastErr error
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			slog.Debug("retrying search request", "url", url, "attempt", attempt, "err", lastErr)
			select {
			case <-ctx.Done():
				return nil, redact.Errorf("GET %s: %w", redact.Safe(url), redact.Safe(ctx.Err()))
			case <-time.After(retryBackoff(attempt)):
			}
		}
		result, retryable, err := execGetOnce[T](ctx, url)
		if err == nil || !retryable {
			return result, err
		}
		lastErr = err
	}
	return nil, redact.Errorf("%w: %w", redact.Safe(ErrUnavailable), redact.Safe(lastErr))
}

// execGetOnce makes a single request. retryable reports whether the error is
// transient (network error, 429, or 5xx).
func execGetOnce[T any](ctx context.Context, url string) (result *T, retryable bool, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, false, redact.Errorf("GET %s: %w", redact.Safe(url), redact.Safe(err))
	}
	req.Header.Set("User-Agent", userAgent)

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		// Don't retry if the caller gave up.
		return nil, ctx.Err() == nil, redact.Errorf("GET %s: %w", redact.Safe(url), redact.Safe(err))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, ctx.Err() == nil, redact.Errorf("GET %s: read response body: %w", redact.Safe(url), redact.Safe(err))
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, false, ErrNotFound
	}
	if response.StatusCode >= 400 {
		retryable := response.StatusCode >= 500 || response.StatusCode == http.StatusTooManyRequests
		return nil, retryable, redact.Errorf("GET %s: unexpected status code %s: %s",
			redact.Safe(url),
			redact.Safe(response.Status),
			redact.Safe(data),
		)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, false, redact.Errorf("GET %s: unmarshal response JSON: %w", redact.Safe(url), redact.Safe(err))
	}
	return &out, false, nil
}
