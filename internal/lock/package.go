// Copyright 2024 Jetify Inc and contributors. All rights reserved.
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
	StorePath             string `json:"store_path,omitempty"`
	outputIsFromStorePath bool
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

func (i *SystemInfo) Output(name string) (Output, error) {
	if i == nil {
		return Output{}, nil
	}

	for _, output := range i.Outputs {
		if output.Name == name {
			return output, nil
		}
	}

	return Output{}, fmt.Errorf("Output %s not found", name)
}

func (i *SystemInfo) DefaultOutputs() []Output {
	if i == nil {
		return nil
	}

	if len(i.Outputs) == 0 {
		return nil
	}

	res := []Output{}
	for _, output := range i.Outputs {
		if output.Default {
			res = append(res, output)
		}
	}
	if len(res) > 0 {
		return res
	}

	// If no default outputs, return the first one
	return []Output{i.Outputs[0]}
}

func (i *SystemInfo) Equals(other *SystemInfo) bool {
	if i == nil || other == nil {
		return i == other
	}

	return slices.Equal(i.Outputs, other.Outputs)
}

// If we have a StorePath and no Outputs, we need to convert to the new format.
// Note: non-empty Outputs may have StorePath alongside it for backwards-compatibility
// but we should ignore that legacy StorePath.
func (i *SystemInfo) addOutputFromLegacyStorePath() {
	if i.StorePath != "" && len(i.Outputs) == 0 {
		i.Outputs = []Output{
			{
				Default: true,
				Name:    "out",
				Path:    i.StorePath,
			},
		}
		i.outputIsFromStorePath = true
	}
}

// ensurePackagesHaveOutputs is used for backwards-compatibility with the old
// lockfile format where each SystemInfo had a StorePath but no Outputs.
func ensurePackagesHaveOutputs(packages map[string]*Package) {
	for _, pkg := range packages {
		for sys, sysInfo := range pkg.Systems {
			sysInfo.addOutputFromLegacyStorePath()
			pkg.Systems[sys] = sysInfo
		}
	}
}
