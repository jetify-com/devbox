package shellgen

import (
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/nix"
)

type flakePackage struct {
	Name      string
	FromStore string
	FromPath  string
}

func newFlakePackage(pkg *nix.Package) (*flakePackage, error) {
	// nixosCacheURL is where we fetch package binaries from
	const nixosCacheURL = "https://cache.nixos.org"

	// flakePackages only support versioned packages that have a system info
	sysInfo := pkg.SystemInfo()
	if sysInfo == nil {
		return nil, nil
	}

	attributePath, err := pkg.PackageAttributePath()
	if err != nil {
		return nil, err
	}

	storeDir := strings.Join([]string{sysInfo.FromHash, sysInfo.StoreName, sysInfo.StoreVersion}, "-")
	return &flakePackage{
		Name:      attributePath,
		FromStore: nixosCacheURL,
		FromPath:  filepath.Join("/nix/store", storeDir),
		// TODO add ToPath:
	}, nil
}

func flakePackages(devbox devboxer) ([]*flakePackage, error) {
	if !featureflag.RemoveNixpkgs.Enabled() {
		return nil, nil
	}

	userInputs := devbox.PackagesAsInputs()
	pluginInputs, err := devbox.PluginManager().PluginInputs(userInputs)
	if err != nil {
		return nil, err
	}
	// As per flakeInputs function comments, we prioritize plugin packages
	// so the php plugin works.
	pkgs := append(pluginInputs, userInputs...)

	result := []*flakePackage{}
	for _, pkg := range pkgs {
		flakePkg, err := newFlakePackage(pkg)
		if err != nil {
			return nil, err
		}
		if flakePkg != nil {
			result = append(result, flakePkg)
		}
	}
	return result, nil
}
