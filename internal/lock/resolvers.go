// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

type devboxProject interface {
	ConfigHash() (string, error)
	ProjectDir() string
}

type resolver interface {
	IsVersionedPackage(pkg string) bool
	Resolve(pkg, version string) (*Package, error)
}

type Locker interface {
	devboxProject
	IsVersionedPackage(pkg string) bool
	Resolve(pkg string) (string, error)
}
