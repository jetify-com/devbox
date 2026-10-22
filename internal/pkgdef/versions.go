// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strings"
)

// ListVersions returns every version the definition's version source knows
// about, unsorted and including pre-releases.
func (c *Client) ListVersions(ctx context.Context, def *Definition) ([]string, error) {
	src := def.Versions
	switch src.Type {
	case VersionsStatic:
		return slices.Clone(src.Versions), nil
	case VersionsHTTPJSON:
		return c.listVersionsHTTPJSON(ctx, def.ResolveURL(src.URL), src.Select)
	case VersionsGitHubReleases:
		return c.listVersionsGitHub(ctx, src)
	case VersionsScript:
		return nil, fmt.Errorf("versions.type %q: %w", src.Type, ErrNotImplemented)
	default:
		return nil, fmt.Errorf("versions.type %q is not supported", src.Type)
	}
}

// ResolveVersion resolves a version constraint ("latest", "1.22", "1.22.3")
// to a concrete version using the definition's version source.
func (c *Client) ResolveVersion(ctx context.Context, def *Definition, requested string) (string, error) {
	available, err := c.ListVersions(ctx, def)
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
	body, err := c.get(ctx, rawURL)
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

func (c *Client) listVersionsGitHub(ctx context.Context, src VersionSource) ([]string, error) {
	owner, repo, _ := strings.Cut(src.Repo, "/")
	stripPrefix := "v"
	if src.StripPrefix != nil {
		stripPrefix = *src.StripPrefix
	}

	// The first page of 100 releases is enough for version resolution in
	// practice; pagination can be added when a definition needs it.
	body, err := c.getGitHubAPI(ctx, "repos/"+owner+"/"+repo+"/releases?per_page=100")
	if err != nil {
		return nil, fmt.Errorf("github releases for %s: %w", src.Repo, err)
	}
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("github releases for %s: response is not JSON: %w", src.Repo, err)
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
	return versions, nil
}
