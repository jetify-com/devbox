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
	"time"

	"github.com/pkg/errors"
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

func (c *client) Search(ctx context.Context, query string) (*SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("query should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "v1/search")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	searchURL := endpoint + "?q=" + url.QueryEscape(query)

	return execGet[SearchResults](ctx, searchURL)
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
