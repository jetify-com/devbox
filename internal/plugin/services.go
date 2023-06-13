// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"fmt"
	"os"

	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/services"
)

func (m *Manager) GetServices(
	pkgs []*nix.Input,
	includes []string,
) (services.Services, error) {
	allSvcs := services.Services{}

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

		svcs, err := conf.Services()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error reading services in plugin \"%s\", skipping", conf.Name)
			continue
		}
		for name, svc := range svcs {
			allSvcs[name] = svc
		}
	}

	return allSvcs, nil
}
