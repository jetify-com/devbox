// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/services"
)

// TODO: this should have PluginManager as receiver so we can build once with
// pkgs, includes, etc
func (m *Manager) GetServices(
	pkgs []*nix.Input,
	includes []string,
) (services.Services, error) {
	svcs := services.Services{}

	allPkgs := append([]*nix.Input(nil), pkgs...)
	for _, include := range includes {
		name, err := m.parseInclude(include)
		if err != nil {
			return nil, err
		}
		allPkgs = append(allPkgs, name)
	}

	for _, pkg := range allPkgs {
		conf, err := getConfigIfAny(pkg, m.ProjectDir())
		if err != nil {
			return nil, err
		}
		if conf == nil {
			continue
		}

		if file, ok := conf.ProcessComposeYaml(); ok {
			svc := services.Service{
				Name:               conf.Name,
				Env:                conf.Env,
				ProcessComposePath: file,
			}
			svcs[conf.Name] = svc
		}

	}
	return svcs, nil
}
