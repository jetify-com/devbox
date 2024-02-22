// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"go.jetpack.io/devbox/internal/devpkg"
)

func (m *Manager) InitHooks(
	pkgs []*devpkg.Package,
	includes []string,
) ([]string, error) {
	hooks := []string{}
	allPkgs := []Includable{}
	for _, pkg := range pkgs {
		allPkgs = append(allPkgs, pkg)
	}
	for _, include := range includes {
		name, err := m.ParseInclude(include)
		if err != nil {
			return nil, err
		}
		allPkgs = append(allPkgs, name)
	}
	for _, pkg := range allPkgs {
		c, err := getConfigIfAny(pkg, m.ProjectDir())
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		hooks = append(hooks, c.InitHook().Cmds...)
	}
	return hooks, nil
}
