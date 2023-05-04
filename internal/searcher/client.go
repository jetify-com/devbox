// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/lock"
)

const searchAPIEndpoint = "https://search.devbox.sh"

func searchHost() string {
	endpoint := searchAPIEndpoint
	if os.Getenv(envir.DevboxSearchHost) != "" {
		endpoint = os.Getenv(envir.DevboxSearchHost)
	}
	return endpoint
}

type client struct {
	endpoint string
}

var cachedClient *client

func Client() *client {
	if cachedClient == nil {
		endpoint, _ := url.JoinPath(searchHost(), "search")
		cachedClient = &client{
			endpoint: endpoint,
		}
	}
	return cachedClient
}

func (c *client) Search(query string) (*SearchResult, error) {
	return execSearch(c.endpoint + "?q=" + url.QueryEscape(query))
}

func (c *client) SearchVersion(query, version string) (*SearchResult, error) {
	return execSearch(
		c.endpoint +
			"?q=" + url.QueryEscape(query) +
			"&v=" + url.QueryEscape(version),
	)
}

func (c *client) Resolve(pkg string) (*lock.Package, error) {
	name, version, _ := strings.Cut(pkg, "@")
	if version == "" {
		return nil, usererr.New("No version specified for %q.", name)
	}
	result, err := c.SearchVersion(name, version)
	if err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, usererr.New("No results found for %q.", pkg)
	}
	return &lock.Package{
		LastModified: result.Results[0].Packages[0].Date,
		Resolved: fmt.Sprintf(
			"github:NixOS/nixpkgs/%s#%s",
			result.Results[0].Packages[0].NixpkgCommit,
			result.Results[0].Packages[0].AttributePath,
		),
		Version: result.Results[0].Packages[0].Version,
	}, nil
}

func execSearch(url string) (*SearchResult, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var result SearchResult
	return &result, json.Unmarshal(data, &result)
}
