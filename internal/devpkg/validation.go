package devpkg

import (
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
	"golang.org/x/exp/slices"
)

func (p *Package) ValidateExists() (bool, error) {
	if p.isVersioned() && p.version() == "" {
		return false, usererr.New("No version specified for %q.", p.Path)
	}

	inCache, err := p.IsInBinaryCache()
	if err != nil {
		return false, err
	}
	if inCache {
		return true, nil
	}

	info, err := p.NormalizedPackageAttributePath()
	return info != "", err
}

func (p *Package) ValidateInstallsOnSystem() (bool, error) {
	u, err := p.urlForInstall()
	if err != nil {
		return false, err
	}
	info := nix.Search(u)
	if len(info) == 0 {
		return false, nil
	}

	platforms := nix.PackagePlatforms(u)

	if len(platforms) == 0 {
		// We're not sure, just return true.
		return true, nil
	}

	currentSystem, err := nix.System()
	if err != nil {
		return false, err
	}

	return slices.Contains(platforms, currentSystem), nil
}
