// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

type NixPackageInfoList []*NixPackageInfo

type NixPackageInfo struct {
	AttributePath string  `json:"attribute_path,omitempty"`
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
	Metadata Metadata `json:"metadata"`
	Results  []Result `json:"results"`
}
