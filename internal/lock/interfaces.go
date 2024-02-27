// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

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

func (d *DummyLocker) Resolve(string) (*Package, error) {
	return nil, nil
}
