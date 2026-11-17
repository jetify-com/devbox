// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// maxPages bounds how many pages of a paginated version source are read.
const maxPages = 20

// githubPerPage is the page size for the GitHub releases API (its maximum).
const githubPerPage = 100

// ListVersions returns every version the definition's version sources
// know about, unsorted and including pre-releases. Paginated sources are
// read up to maxPages.
func (c *Client) ListVersions(ctx context.Context, def *Definition) ([]string, error) {
	return c.listVersions(ctx, def, nil)
}

// listVersions lists the versions of every source and returns their union.
// When enough is not nil, a paginated source stops being read as soon as
// enough reports true for the versions read from it so far.
func (c *Client) listVersions(ctx context.Context, def *Definition, enough func([]string) bool) ([]string, error) {
	var all []string
	for _, src := range def.Versions {
		versions, err := c.listSource(ctx, def, src, enough)
		if err != nil {
			return nil, err
		}
		all = append(all, versions...)
	}
	return all, nil
}

func (c *Client) listSource(ctx context.Context, def *Definition, src VersionSource, enough func([]string) bool) ([]string, error) {
	switch src.Type {
	case VersionsStatic:
		return slices.Clone(src.Versions), nil
	case VersionsHTTPJSON:
		if !strings.Contains(src.URL, ".Page") {
			return c.listVersionsHTTPJSON(ctx, def.ResolveURL(src.URL), src.Select)
		}
		return readPages(enough, 0, func(page int) ([]string, bool, error) {
			rawURL, err := Render(src.URL, Vars{Page: page})
			if err != nil {
				return nil, false, err
			}
			versions, err := c.listVersionsHTTPJSON(ctx, def.ResolveURL(rawURL), src.Select)
			if errors.Is(err, ErrNotFound) && page > 0 {
				return nil, false, nil
			}
			return versions, len(versions) > 0, err
		})
	case VersionsGitHubReleases:
		return readPages(enough, 1, func(page int) ([]string, bool, error) {
			return c.listVersionsGitHub(ctx, src, page)
		})
	case VersionsScript:
		return nil, fmt.Errorf("versions.type %q: %w", src.Type, ErrNotImplemented)
	default:
		return nil, fmt.Errorf("versions.type %q is not supported", src.Type)
	}
}

// readPages reads pages starting at first until fetch reports there are no
// more, maxPages have been read, or enough is satisfied.
func readPages(enough func([]string) bool, first int, fetch func(page int) ([]string, bool, error)) ([]string, error) {
	var all []string
	for page := first; page < first+maxPages; page++ {
		versions, more, err := fetch(page)
		if err != nil {
			return nil, err
		}
		all = append(all, versions...)
		if !more || (enough != nil && enough(all)) {
			break
		}
	}
	return all, nil
}

// ResolveVersion resolves a version constraint ("latest", "1.22", "1.22.3")
// to a concrete version using the definition's version sources. Paginated
// sources are read until a page yields a match, so they should list newest
// versions first.
func (c *Client) ResolveVersion(ctx context.Context, def *Definition, requested string) (string, error) {
	enough := func(versions []string) bool {
		_, err := MatchVersion(versions, requested)
		return err == nil
	}
	available, err := c.listVersions(ctx, def, enough)
	if err != nil {
		return "", fmt.Errorf("%s: listing versions: %w", def.Name, err)
	}
	version, err := MatchVersion(available, requested)
	if err != nil {
		return "", fmt.Errorf("%s: %w", def.Name, err)
	}
	return version, nil
}

func (c *Client) listVersionsHTTPJSON(ctx context.Context, rawURL, selectExpr string) ([]string, error) {
	code, err := compileJQ(selectExpr)
	if err != nil {
		return nil, err
	}
	body, err := c.getCached(ctx, rawURL)
	if err != nil {
		return nil, err
	}
	var input any
	if err := json.Unmarshal(body, &input); err != nil {
		return nil, fmt.Errorf("%s: response is not JSON: %w", rawURL, err)
	}
	return runJQ(code, selectExpr, input)
}

// githubRelease is the subset of the GitHub releases API response we need.
type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
}

// listVersionsGitHub lists one page of a repository's releases, newest
// first, and reports whether there may be more pages.
func (c *Client) listVersionsGitHub(ctx context.Context, src VersionSource, page int) ([]string, bool, error) {
	owner, repo, _ := strings.Cut(src.Repo, "/")
	stripPrefix := "v"
	if src.StripPrefix != nil {
		stripPrefix = *src.StripPrefix
	}

	apiPath := "repos/" + owner + "/" + repo + "/releases?per_page=" + strconv.Itoa(githubPerPage)
	if page > 1 {
		apiPath += "&page=" + strconv.Itoa(page)
	}
	body, err := c.getGitHubAPI(ctx, apiPath)
	if err != nil {
		return nil, false, fmt.Errorf("github releases for %s: %w", src.Repo, err)
	}
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, false, fmt.Errorf("github releases for %s: response is not JSON: %w", src.Repo, err)
	}

	versions := make([]string, 0, len(releases))
	for _, rel := range releases {
		if rel.Draft || rel.Prerelease {
			continue
		}
		tag := rel.TagName
		if stripPrefix != "" {
			if !strings.HasPrefix(tag, stripPrefix) {
				continue
			}
			tag = strings.TrimPrefix(tag, stripPrefix)
		}
		if tag != "" {
			versions = append(versions, tag)
		}
	}
	return versions, len(releases) == githubPerPage, nil
}
