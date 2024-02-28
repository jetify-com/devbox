// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import "strings"

type devboxProject interface {
	ConfigHash() (string, error)
	NixPkgsCommitHash() string
	PackageNames() []string
	ProjectDir() string
}

type Locker interface {
	Get(string) *Package
	LegacyNixpkgsPath(string) string
	ProjectDir() string
	Resolve(string) (*Package, error)
}

type DummyLocker struct {
	ProjectDirVal string
}

func (d *DummyLocker) Get(string) *Package {
	return nil
}

func (d *DummyLocker) LegacyNixpkgsPath(string) string {
	return ""
}

func (d *DummyLocker) ProjectDir() string {
	return d.ProjectDirVal
}

func (d *DummyLocker) Resolve(s string) (*Package, error) {
	a, _, _ := strings.Cut(s, "@")
	return &Package{
		Resolved: "github:NixOS/nixpkgs/75a52265bda7fd25e06e3a67dee3f0354e73243c#" + a,
		Source:   nixpkgSource,
	}, nil
}
