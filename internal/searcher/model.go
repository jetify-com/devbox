// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

type SearchResults struct {
	NumResults int       `json:"num_results"`
	Packages   []Package `json:"packages,omitempty"`
}

type Package struct {
	Name        string           `json:"name"`
	NumVersions int              `json:"num_versions"`
	Versions    []PackageVersion `json:"versions,omitempty"`
}

type PackageVersion struct {
	PackageInfo

	Name    string                 `json:"name"`
	Systems map[string]PackageInfo `json:"systems,omitempty"`
}

type PackageInfo struct {
	ID           int      `json:"id"`
	CommitHash   string   `json:"commit_hash"`
	System       string   `json:"system"`
	LastUpdated  int      `json:"last_updated"`
	StoreHash    string   `json:"store_hash"`
	StoreName    string   `json:"store_name"`
	StoreVersion string   `json:"store_version"`
	MetaName     string   `json:"meta_name"`
	MetaVersion  []string `json:"meta_version"`
	AttrPaths    []string `json:"attr_paths"`
	Version      string   `json:"version"`
}
