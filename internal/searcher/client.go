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
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
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

func (c *client) Search(query string, options ...SearchOption) (*SearchResult, error) {
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

// SearchOption returns a string for query to be appended to the endpoint
type SearchOption func() string

func WithVersion(version string) SearchOption {
	return func() string {
		return "&v=" + url.QueryEscape(version)
	}
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

	searchVersion := result.Results[0].Packages[0].Version
	sysInfos := map[string]*lock.SystemInfo{}
	if featureflag.RemoveNixpkgs.Enabled() {
		// we use searchVersion instead of version so that "latest" is resolved
		// to a concrete version before we get the package's system info
		sysInfosQueried, err := c.resolvePackageSystemInfoIfAny(name, searchVersion)
		if err != nil {
			return nil, err
		}
		if sysInfosQueried != nil {
			sysInfos = sysInfosQueried
		}
	}
	return &lock.Package{
		LastModified: result.Results[0].Packages[0].Date,
		Resolved: fmt.Sprintf(
			"github:NixOS/nixpkgs/%s#%s",
			result.Results[0].Packages[0].NixpkgCommit,
			result.Results[0].Packages[0].AttributePath,
		),
		Version: searchVersion,
		Systems: sysInfos,
	}, nil
}

// resolvePackageSystemInfoIfAny is temporary, until the search API returns
// the "system info" like the store-hash. This uses the /pkg api that is
// for nixhub.io as a temporary workaround.
func (c *client) resolvePackageSystemInfoIfAny(pkgName, version string) (map[string]*lock.SystemInfo, error) {
	packageResults, err := execPackageQuery(c.host, pkgName)
	if err != nil {
		return nil, err
	}

	var ok bool
	result, ok := lo.Find(
		packageResults, func(result *PackageResult) bool { return result.Version == version })
	if !ok {
		return nil, nil
	}

	systemInfos := map[string]*lock.SystemInfo{}
	for sysName, sysInfo := range result.Systems {
		systemInfos[sysName] = &lock.SystemInfo{
			System:       sysName,
			FromHash:     sysInfo.StoreHash,
			StoreName:    sysInfo.StoreName,
			StoreVersion: sysInfo.StoreVersion,
		}
	}
	return systemInfos, nil
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
