// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import "go.jetpack.io/devbox/nix/flake"

type devboxProject interface {
	ConfigHash() (string, error)
	Stdenv() flake.Ref
	AllPackageNamesIncludingRemovedTriggerPackages() []string
	ProjectDir() string
}

type Locker interface {
	Get(string) *Package
	Stdenv() flake.Ref
	ProjectDir() string
	Resolve(string) (*Package, error)
}
