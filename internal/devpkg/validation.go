package devpkg

import (
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
)

func (p *Package) ValidateExists() (bool, error) {
	if p.IsRunX() {
		// TODO implement runx validation
		return true, nil
	}
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
	info, _ := nix.Search(u)
	if len(info) == 0 {
		return false, nil
	}
	if out, err := nix.Eval(u); err != nil &&
		strings.Contains(string(out), "is not available on the requested hostPlatform") {
		return false, nil
	}
	// There's other stuff that may cause this evaluation to fail, but we don't
	// want to handle all of them here. (e.g. unfree packages)
	return true, nil
}
