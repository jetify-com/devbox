// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/redact"
)

const searchAPIEndpoint = "https://search.devbox.sh"

var ErrNotFound = errors.New("Not found")

type client struct {
	host string
}

func Client() *client {
	return &client{
		host: envir.GetValueOrDefault(envir.DevboxSearchHost, searchAPIEndpoint),
	}
}

func (c *client) Search(query string) (*SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("query should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "v1/search")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	searchURL := endpoint + "?q=" + url.QueryEscape(query)

	return execGet[SearchResults](context.TODO(), searchURL)
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

func execGet[T any](ctx context.Context, url string) (*T, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, redact.Errorf("GET %s: %w", redact.Safe(url), redact.Safe(err))
	}
	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, redact.Errorf("GET %s: %w", redact.Safe(url), redact.Safe(err))
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, redact.Errorf("GET %s: read respoonse body: %w", redact.Safe(url), redact.Safe(err))
	}
	if response.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if response.StatusCode >= 400 {
		return nil, redact.Errorf("GET %s: unexpected status code %s: %s",
			redact.Safe(url),
			redact.Safe(response.Status),
			redact.Safe(data),
		)
	}
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, redact.Errorf("GET %s: unmarshal response JSON: %w", redact.Safe(url), redact.Safe(err))
	}
	return &result, nil
}
