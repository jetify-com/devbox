// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// flakeInputs returns a list of flake inputs for the top level flake.nix
// created by devbox. We map packages to the correct flake and attribute path
// and group flakes by URL to avoid duplication. All inputs should be locked
// i.e. have a commit hash and always resolve to the same package/version.
// Note: inputs returned by this function include plugin packages. (php only for now)
// It's not entirely clear we always want to add plugin packages to the top level
func (d *Devbox) flakeInputs() ([]*plansdk.FlakeInput, error) {
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
		AttributePath, err := pkg.PackageAttributePath()
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

	return PickByKeysSorted(inputs, order), nil
}

// TODO: move this to a util package
func PickByKeysSorted[K comparable, V any](in map[K]V, keys []K) []V {
	out := make([]V, len(keys))
	for i, key := range keys {
		out[i] = in[key]
	}
	return out
}
