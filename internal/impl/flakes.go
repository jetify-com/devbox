// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"runtime/trace"
	"strings"

	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/goutil"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// flakeInputs returns a list of flake inputs for the top level flake.nix
// created by devbox. We map packages to the correct flake and attribute path
// and group flakes by URL to avoid duplication. All inputs should be locked
// i.e. have a commit hash and always resolve to the same package/version.
// Note: inputs returned by this function include plugin packages. (php only for now)
// It's not entirely clear we always want to add plugin packages to the top level
func (d *Devbox) flakeInputs(ctx context.Context) ([]*plansdk.FlakeInput, error) {
	defer trace.StartRegion(ctx, "flakeInputs").End()

	inputs := map[string]*plansdk.FlakeInput{}

	userPackages := d.packagesAsInputs()
	pluginPackages, err := d.pluginManager.PluginPackages(userPackages)
	if err != nil {
		return nil, err
	}

	order := []string{}
	// We prioritize plugin packages so that the php plugin works. Not sure
	// if this is behavior we want for user plugins. We may need to add an optional
	// priority field to the config.
	for _, pkg := range append(pluginPackages, userPackages...) {
		AttributePath, err := pkg.FullPackageAttributePath()
		if err != nil {
			return nil, err
		}
		if input, ok := inputs[pkg.URLForInput()]; !ok {
			order = append(order, pkg.URLForInput())
			inputs[pkg.URLForInput()] = &plansdk.FlakeInput{
				Name:     pkg.InputName(),
				URL:      pkg.URLForInput(),
				Packages: []string{AttributePath},
			}
		} else {
			input.Packages = lo.Uniq(
				append(inputs[pkg.URLForInput()].Packages, AttributePath),
			)
		}
	}

	return goutil.PickByKeysSorted(inputs, order), nil
}

// getLocalFlakesDirs searches packages and returns list of directories
// of local flakes that are mentioned in config.
// e.g., path:./my-flake#packageName -> ./my-flakes
func (d *Devbox) getLocalFlakesDirs() []string {
	localFlakeDirs := []string{}

	// searching through installed packages to get location of local flakes
	for _, pkg := range d.Packages() {
		// filtering local flakes packages
		if strings.HasPrefix(pkg, "path:") {
			pkgDirAndName, _ := strings.CutPrefix(pkg, "path:")
			pkgDir := strings.Split(pkgDirAndName, "#")[0]
			localFlakeDirs = append(localFlakeDirs, pkgDir)
		}
	}
	return localFlakeDirs
}
