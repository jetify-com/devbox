package impl

import (
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

func (d *Devbox) flakeInputs() []*plansdk.FlakeInput {
	inputs := map[string]*plansdk.FlakeInput{}
	for _, p := range d.cfg.MergedPackages(d.writer) {
		pkg := nix.InputFromString(p, d.projectDir)
		if pkg.IsFlake() {
			if input, ok := inputs[pkg.URLWithoutFragment()]; !ok {
				inputs[pkg.URLWithoutFragment()] = &plansdk.FlakeInput{
					Name:     pkg.Name(),
					URL:      pkg.URLWithoutFragment(),
					Packages: []string{pkg.Package()},
				}
			} else {
				input.Packages = lo.Uniq(
					append(inputs[pkg.URLWithoutFragment()].Packages, pkg.Package()),
				)
			}
		}
	}

	return lo.Values(inputs)
}
