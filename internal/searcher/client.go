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
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
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

func (c *client) Search(query string, options ...func() string) (*SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("query should not be empty")
	}

	endpoint, err := url.JoinPath(c.host, "search")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	searchURL := endpoint + "?q=" + url.QueryEscape(query)

	for _, op := range options {
		searchURL += op()
	}

	return execSearch(searchURL)
}

func WithVersion(version string) func() string {
	return func() string {
		return "&v=" + url.QueryEscape(version)
	}
}

func (c *client) PackageInfo(pkgName string) ([]*PackageResult, error) {
	return execPackageQuery(c.host, pkgName)
}

func (c *client) Resolve(pkg string) (*lock.Package, error) {
	name, version, _ := devpkg.ParseVersionedPackage(pkg)
	if version == "" {
		return nil, usererr.New("No version specified for %q.", name)
	}
	result, err := c.Search(name, WithVersion(version))
	if err != nil {
		return nil, err
	}
	if len(result.Results) == 0 {
		return nil, nix.ErrPackageNotFound
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
	fmt.Printf("queried url %s\n and got response:\n %+v\n", url, string(data))
	var result SearchResult
	return &result, json.Unmarshal(data, &result)
}

func execPackageQuery(endpoint, pkgName string) ([]*PackageResult, error) {
	url, err := url.JoinPath(endpoint, "pkg", pkgName)
	if err != nil {
		return nil, err
	}
	response, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	data, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	var result []*PackageResult
	return result, json.Unmarshal(data, &result)
}
