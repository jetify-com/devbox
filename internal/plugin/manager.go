// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
)

type Manager struct {
	devboxProject

	lockfile *lock.File
}

type devboxProject interface {
	Packages() []string
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

func (m *Manager) PluginInputs(inputs []*nix.Package) ([]*nix.Package, error) {
	result := []*nix.Package{}
	for _, input := range inputs {
		config, err := getConfigIfAny(input, m.ProjectDir())
		if err != nil {
			return nil, err
		} else if config == nil {
			continue
		}
		result = append(result, nix.PackageFromStrings(config.Packages, m.lockfile)...)
	}
	return result, nil
}
