// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

type NixPackageInfoList []*NixPackageInfo

type NixPackageInfo struct {
	AttributePath string  `json:"attribute_path,omitempty"`
	Date          string  `json:"date,omitempty"`
	NixpkgCommit  string  `json:"nixpkg_commit,omitempty"`
	PName         string  `json:"pname,omitempty"`
	Version       string  `json:"version,omitempty"`
	Score         float64 `json:"score,omitempty"`
}

type Result struct {
	Name     string             `json:"name"`
	Packages NixPackageInfoList `json:"packages"`
	Score    float64            `json:"score"`
}

type Metadata struct {
	TotalResults int `json:"total_results"` // This will undercount if there are more than 1000 results per key
}

type SearchResult struct {
	Metadata    Metadata `json:"metadata"`
	Results     []Result `json:"results"`
	Suggestions []Result `json:"suggestions"`
}

// Package api:
// https://search.devbox.sh/pkg/<name>

type PackageResult struct {
	Summary  string                `json:"summary"`
	Homepage string                `json:"homepage"`
	License  string                `json:"license"`
	Name     string                `json:"name"`
	Version  string                `json:"version"`
	Systems  map[string]SystemInfo `json:"systems"`
}

type SystemInfo struct {
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
}
