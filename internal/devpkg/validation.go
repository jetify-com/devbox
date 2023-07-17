package devpkg

import (
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
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

// ValidateInstallsOnSystem returns true if the package can be installed
// on system. Unfortunately, this is not comprehensive. Specifically, nix
// checks meta.platforms and meta.badPlatforms like so:
// https://github.com/NixOS/nixpkgs/blob/4a716c50feec750263bee793ceb571be536dff19/lib/meta.nix#L95-L106
// We could copy that logic here, but it may make more sense to use a nix expression
// One issue I ran into using nix was that it was too slow because it has to
// download nixpkgs and it was not using the version already prefetched.
func (p *Package) ValidateInstallsOnSystem() (bool, error) {
	u, err := p.urlForInstall()
	if err != nil {
		return false, err
	}
	info, _ := nix.Search(u)
	return len(info) > 0, nil
}
