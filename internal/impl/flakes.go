package impl

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) flakeInputs() []plansdk.FlakeInput {
	inputs := map[string]plansdk.FlakeInput{}
	for _, p := range d.cfg.MergedPackages(d.writer) {
		pkg := nix.InputFromString(p, d.projectDir)
		if pkg.IsFlake() {
			if input, ok := inputs[pkg.Name()]; !ok {
				inputs[pkg.Name()] = plansdk.FlakeInput{
					Name:     pkg.Name(),
					URL:      pkg.URLWithoutFragment(),
					Packages: pkg.Packages(),
				}
			} else {
				input.Packages = lo.Uniq(
					append(inputs[pkg.Name()].Packages, pkg.Packages()...),
				)
			}
		}
	}

	return lo.Values(inputs)
}
