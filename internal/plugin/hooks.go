// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import "go.jetpack.io/devbox/internal/nix"

func InitHooks(pkgs []*nix.Package, projectDir string) ([]string, error) {
	hooks := []string{}
	for _, pkg := range pkgs {
		c, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		hooks = append(hooks, c.Shell.InitHook.Cmds...)
	}
	return hooks, nil
}
