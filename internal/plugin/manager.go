// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"go.jetpack.io/devbox/internal/lock"
)

type Manager struct {
	devboxProject

	lockfile *lock.File
}

type devboxProject interface {
	PackageNames() []string
	ProjectDir() string
}

type managerOption func(*Manager)

func NewManager(opts ...managerOption) *Manager {
	m := &Manager{}
	m.ApplyOptions(opts...)
	return m
}

func WithLockfile(lockfile *lock.File) managerOption {
	return func(m *Manager) {
		m.lockfile = lockfile
	}
}

func WithDevbox(provider devboxProject) managerOption {
	return func(m *Manager) {
		m.devboxProject = provider
	}
}

func (m *Manager) ApplyOptions(opts ...managerOption) {
	for _, opt := range opts {
		opt(m)
	}
}
