// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

type devboxProject interface {
	ConfigHash() (string, error)
	NixPkgsCommitHash() string
	Packages() []string
	ProjectDir() string
}

type Locker interface {
	LegacyNixpkgsPath(string) string
	ProjectDir() string
	Resolve(string) (*Package, error)
}
