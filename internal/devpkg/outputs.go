package devpkg

import (
	"strings"

	"go.jetify.com/devbox/internal/boxcli/usererr"
)

type Output struct {
	Name     string
	CacheURI string
}

// outputs are the nix package outputs
type outputs struct {
	selectedNames []string
	defaultNames  []string
}

func (out *outputs) GetNames(pkg *Package) ([]string, error) {
	if len(out.selectedNames) > 0 {
		return out.selectedNames, nil
	}

	// else, get the default outputs from the lockfile
	// if we haven't already
	if out.defaultNames == nil {
		if err := out.initDefaultNames(pkg); err != nil {
			return []string{}, err
		}
	}
	return out.defaultNames, nil
}

// initDefaultNames initializes the defaultNames field of the Outputs object.
// We run this lazily (rather than eagerly in initOutputs) because it depends on the Package,
// and initOutputs is called from the Package constructor, so cannot depend on Package.
func (out *outputs) initDefaultNames(pkg *Package) error {
	sysInfo, err := pkg.sysInfoIfExists()
	if err != nil {
		// Provide better error for npm packages not found in Nixpkgs
		errMsg := strings.TrimSpace(err.Error())
		if strings.Contains(errMsg, "package not found") && strings.HasPrefix(pkg.Raw, "nodePackages.") {
			npmPkg := strings.TrimPrefix(pkg.Raw, `nodePackages."`)
			npmPkg = strings.TrimSuffix(npmPkg, `"`)
			return usererr.New(
				"npm package %s was not found in Nixpkgs.\n"+
					"This may mean the package isn't packaged yet. Check https://search.nixos.org/packages for availability.\n"+
					"As a workaround, you can install it manually in your devbox shell using npm after running: npm config set prefix '~/.npm-global'",
				npmPkg,
			)
		}
		return err
	}

	out.defaultNames = []string{}
	if sysInfo == nil {
		return nil
	}

	for _, output := range sysInfo.DefaultOutputs() {
		out.defaultNames = append(out.defaultNames, output.Name)
	}
	return nil
}
