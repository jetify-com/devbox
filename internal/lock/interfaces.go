// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
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
