// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) flakeInputs() []*plansdk.FlakeInput {
	inputs := map[string]*plansdk.FlakeInput{}
	for _, p := range d.packages() {
		pkg := nix.InputFromString(p, d.lockfile)
		AttributePath, err := pkg.PackageAttributePath()
		if err != nil {
			panic(err)
		}
		if input, ok := inputs[pkg.URLForInput()]; !ok {
			inputs[pkg.URLForInput()] = &plansdk.FlakeInput{
				Name:     pkg.Name(),
				URL:      pkg.URLForInput(),
				Packages: []string{AttributePath},
			}
		} else {
			input.Packages = lo.Uniq(
				append(inputs[pkg.URLForInput()].Packages, AttributePath),
			)
		}
	}

	return lo.Values(inputs)
}
