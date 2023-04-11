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
			AttributePath, err := pkg.PackageAttributePath()
			if err != nil {
				panic(err)
			}
			if input, ok := inputs[pkg.URLWithoutFragment()]; !ok {
				inputs[pkg.URLWithoutFragment()] = &plansdk.FlakeInput{
					Name:     pkg.Name(),
					URL:      pkg.URLWithoutFragment(),
					Packages: []string{AttributePath},
				}
			} else {
				input.Packages = lo.Uniq(
					append(inputs[pkg.URLWithoutFragment()].Packages, AttributePath),
				)
			}
		}
	}

	return lo.Values(inputs)
}
