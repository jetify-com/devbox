// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

const (
	nixpkgSource       string = "nixpkg"
	devboxSearchSource string = "devbox-search"
)

type Package struct {
	AllowInsecure bool   `json:"allow_insecure,omitempty"`
	LastModified  string `json:"last_modified,omitempty"`
	PluginVersion string `json:"plugin_version,omitempty"`
	Resolved      string `json:"resolved,omitempty"`
	Source        string `json:"source,omitempty"`
	Version       string `json:"version,omitempty"`
	// Systems is keyed by the system name
	Systems map[string]*SystemInfo `json:"systems,omitempty"`
}

type SystemInfo struct {
	// StorePath is the input-addressed path for the nix package in /nix/store
	// It is the cache key in the Binary Cache Store (cache.nixos.org)
	// It is of the form /nix/store/<hash>-<name>-<version>
	// <name> may be different from the canonicalName so we store the full store path.
	StorePath string `json:"store_path,omitempty"`
}

func (p *Package) GetSource() string {
	if p == nil {
		return ""
	}
	return p.Source
}

func (p *Package) IsAllowInsecure() bool {
	if p == nil {
		return false
	}
	return p.AllowInsecure
}

func (i *SystemInfo) Equals(other *SystemInfo) bool {
	if i == nil || other == nil {
		return i == other
	}
	return *i == *other
}
