// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"time"

	"go.jetify.com/devbox/nix/flake"
)

type SearchResultsV2 struct {
	Query        string           `json:"query"`
	TotalResults int              `json:"total_results"`
	Results      []SearchResultV2 `json:"results"`
}

type SearchResultV2 struct {
	Name        string    `json:"name"`
	Summary     string    `json:"summary"`
	LastUpdated time.Time `json:"last_updated"`

	// Releases is nil when the search service doesn't support the releases
	// parameter, and the package's releases must come from /v2/pkg instead.
	Releases []SearchRelease `json:"releases"`
}

type SearchRelease struct {
	Version string   `json:"version"`
	Systems []string `json:"systems"`
}

type PackageV2 struct {
	Name        string    `json:"name"`
	Summary     string    `json:"summary"`
	HomepageURL string    `json:"homepage_url"`
	License     string    `json:"license"`
	Releases    []Release `json:"releases"`
}

type Release struct {
	Version     string     `json:"version"`
	LastUpdated time.Time  `json:"last_updated"`
	Platforms   []Platform `json:"platforms"`
}

type Platform struct {
	Arch          string    `json:"arch"`
	OS            string    `json:"os"`
	AttributePath string    `json:"attribute_path"`
	CommitHash    string    `json:"commit_hash"`
	Date          time.Time `json:"date"`
	Outputs       []Output  `json:"outputs"`
}

type Output struct {
	Name    string `json:"name"`
	Path    string `json:"path"`
	Default bool   `json:"default"`
	Nar     string `json:"nar"`
}

type SearchResults struct {
	NumResults int       `json:"num_results"`
	Packages   []Package `json:"packages,omitempty"`
}

type Package struct {
	Name     string          `json:"name"`
	Versions []SearchRelease `json:"versions,omitempty"`
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
	Summary      string   `json:"summary"`
}

// ResolveResponse is a response from the /v2/resolve endpoint.
type ResolveResponse struct {
	// Name is the resolved name of the package. For packages that are
	// identifiable by multiple names or attribute paths, this is the
	// "canonical" name.
	Name string `json:"name"`

	// Version is the resolved package version.
	Version string `json:"version"`

	// Summary is a short package description.
	Summary string `json:"summary,omitempty"`

	// Systems contains information about the package that can vary across
	// systems. It will always have at least one system. The keys match a
	// Nix system identifier (aarch64-darwin, x86_64-linux, etc.).
	Systems map[string]struct {
		// FlakeInstallable is a Nix installable that specifies how to
		// install the resolved package version.
		//
		// [Nix installable]: https://nixos.org/manual/nix/stable/command-ref/new-cli/nix#installables
		FlakeInstallable flake.Installable `json:"flake_installable"`

		// LastUpdated is the timestamp of the most recent change to the
		// package.
		LastUpdated time.Time `json:"last_updated"`

		// Outputs provides additional information about the Nix store
		// paths that this package installs. This field is not available
		// for some (especially older) packages.
		Outputs []struct {
			// Name is the output's name. Nix appends the name to
			// the output's store path unless it's the default name
			// of "out". Output names can be anything, but
			// conventionally they follow the various "make install"
			// directories such as "bin", "lib", "src", "man", etc.
			Name string `json:"name,omitempty"`

			// Path is the absolute store path (with the /nix/store/
			// prefix) of the output.
			Path string `json:"path,omitempty"`

			// Default indicates if Nix installs this output by
			// default.
			Default bool `json:"default,omitempty"`

			// NAR is set to the package's NAR archive URL when the
			// output exists in the cache.nixos.org binary cache.
			NAR string `json:"nar,omitempty"`
		} `json:"outputs,omitempty"`
	} `json:"systems"`
}
