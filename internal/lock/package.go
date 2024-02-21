// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"fmt"
	"slices"
)

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

	// NOTE: if you add more fields, please update SyncLockfiles
}

type SystemInfo struct {
	Outputs []Output `json:"outputs,omitempty"`

	// Legacy Format
	StorePath string `json:"store_path,omitempty"`
}

// Output refers to a nix package output. This struct is derived from searcher.Output
type Output struct {
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

// Useful for debugging when we print the struct
func (i *SystemInfo) String() string {
	return fmt.Sprintf("%+v", *i)
}

func (i *SystemInfo) DefaultStorePath() string {
	if i == nil || len(i.Outputs) == 0 {
		return ""
	}

	for _, output := range i.Outputs {
		if output.Default {
			return output.Path
		}
	}

	return i.Outputs[0].Path
}

func (i *SystemInfo) Equals(other *SystemInfo) bool {
	if i == nil || other == nil {
		return i == other
	}

	return slices.Equal(i.Outputs, other.Outputs)
}

// ensurePackagesHaveOutputs is used for backwards-compatibility with the old
// lockfile format where each SystemInfo had a StorePath but no Outputs.
func ensurePackagesHaveOutputs(packages map[string]*Package) {
	for _, pkg := range packages {
		for sys, sysInfo := range pkg.Systems {
			// If we have a StorePath and no Outputs, we need to convert to the new format.
			// Note: for a non-empty StorePath, Outputs should be empty, but being cautious.
			if sysInfo.StorePath != "" && len(sysInfo.Outputs) == 0 {
				pkg.Systems[sys].Outputs = []Output{
					{
						Default: true,
						Name:    "out",
						Path:    sysInfo.StorePath,
					},
				}
			}
		}
	}
}
