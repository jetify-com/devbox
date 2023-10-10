// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/devpkg"
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

// ProcessPluginPackages adds and removes packages as indicated by plugins
func (m *Manager) ProcessPluginPackages(
	userPackages []*devpkg.Package,
) ([]*devpkg.Package, error) {
	pluginPackages := []*devpkg.Package{}
	packagesToRemove := []*devpkg.Package{}
	for _, pkg := range userPackages {
		config, err := getConfigIfAny(pkg, m.ProjectDir())
		if err != nil {
			return nil, err
		} else if config == nil {
			continue
		}
		pluginPackages = append(
			pluginPackages,
			devpkg.PackageFromStrings(config.Packages, m.lockfile)...,
		)
		if config.RemoveTriggerPackage {
			packagesToRemove = append(packagesToRemove, pkg)
		}
	}

	netUserPackages, _ := lo.Difference(userPackages, packagesToRemove)
	// We prioritize plugin packages so that the php plugin works. Not sure
	// if this is behavior we want for user plugins. We may need to add an optional
	// priority field to the config.
	return append(pluginPackages, netUserPackages...), nil
}
