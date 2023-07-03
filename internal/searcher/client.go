// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/searcher/model"
)

const searchAPIEndpoint = "https://search.devbox.sh"

type client struct {
	host string
}

func Client() *client {
	return &client{
		host: envir.GetValueOrDefault(envir.DevboxSearchHost, searchAPIEndpoint),
	}
}

func (c *client) Search(query string) (*model.SearchResults, error) {
	if query == "" {
		return nil, fmt.Errorf("query should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "v1/search")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	searchURL := endpoint + "?q=" + url.QueryEscape(query)

	return execGet[model.SearchResults](searchURL)
}

func (c *client) Resolve(name, version string) (*model.PackageVersion, error) {
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

	return execGet[model.PackageVersion](searchURL)
}

func execGet[T any](url string) (*T, error) {
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	var result T
	return &result, json.Unmarshal(data, &result)
}
