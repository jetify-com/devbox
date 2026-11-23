// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/nix/flake"
)

// noProjectLocker is a [lock.Locker] for commands that inspect packages
// outside of a devbox project. It has no lockfile, so it never resolves
// anything.
type noProjectLocker struct {
	dir string
}

var _ lock.Locker = noProjectLocker{}

func (noProjectLocker) Get(string) *lock.Package { return nil }

func (noProjectLocker) Stdenv() flake.Ref { return flake.Ref{} }

func (l noProjectLocker) ProjectDir() string { return l.dir }

func (noProjectLocker) Resolve(pkg string) (*lock.Package, error) {
	return nil, usererr.New("cannot resolve %q outside of a devbox project", pkg)
}
