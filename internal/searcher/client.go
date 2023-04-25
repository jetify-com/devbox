// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"

	"go.jetpack.io/devbox/internal/env"
)

const searchAPIEndpoint = "https://search.devbox.sh"

func searchHost() string {
	endpoint := searchAPIEndpoint
	if os.Getenv(env.DevboxSearchHost) != "" {
		endpoint = os.Getenv(env.DevboxSearchHost)
	}
	return endpoint
}

type client struct {
	endpoint string
}

func NewClient() *client {
	endpoint, _ := url.JoinPath(searchHost(), "search")
	return &client{
		endpoint: endpoint,
	}
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
