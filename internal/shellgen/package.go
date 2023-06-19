package shellgen

import (
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/searcher"
)

type Package struct {
	Name string // canonical name?

	// FetchClosureArgs is keyed by System
	FetchClosureArgs map[string]FetchClosureArgs
}

// Arguments for nix's builtins.FetchClosure
// https://nixos.org/manual/nix/stable/language/builtins.html#builtins-fetchClosure
type FetchClosureArgs struct {
	System    string
	FromStore string
	FromPath  string
	ToPath    string
}

func flakePackages(devbox devboxer) ([]*Package, error) {
	packages := []*Package{}

	// query the search API and get a parsed response
	// TODO savil. move this to the lockfile, and query the lockfile instead.
	// The search api should never be on the critical path.
	client := searcher.Client()
	for _, in := range devbox.PackagesAsInputs() {
		// TODO savil: handle non-canonical names
		results, err := client.PackageInfo(in.CanonicalName())
		if err != nil {
			return nil, err
		}
		packages = append(packages, NewPackage(results, in))
	}
	return packages, nil
}

func NewPackage(results []*searcher.PackageResult, input *nix.Input) *Package {
	inVersion := input.Version()
	if inVersion == "" {
		return nil
	}
	result, ok := lo.Find(results, func(result *searcher.PackageResult) bool {
		return result.Version == inVersion
	})
	if !ok {
		return nil
	}

	// nixosCacheURL is where we fetch package binaries from
	const nixosCacheURL = "https://cache.nixos.org"

	pkg := &Package{
		Name:             input.CanonicalName(),
		FetchClosureArgs: map[string]FetchClosureArgs{},
	}
	for _, sysInfo := range result.Systems {
		storeDir := strings.Join([]string{sysInfo.StoreHash, sysInfo.StoreName, sysInfo.StoreVersion}, "-")
		fetchClosureArg := FetchClosureArgs{
			System:    sysInfo.System,
			FromStore: nixosCacheURL,
			FromPath:  filepath.Join("/nix/store", storeDir),
			// ToPath:    "TODO",
		}
		pkg.FetchClosureArgs[sysInfo.System] = fetchClosureArg
	}

	// ensure x86-64_darwin platform exists
	if _, ok = pkg.FetchClosureArgs["x86_64-darwin"]; !ok {
		if linuxFCA, ok := pkg.FetchClosureArgs["x86_64-linux"]; ok {
			pkg.FetchClosureArgs["x86_64-darwin"] = FetchClosureArgs{
				System:    linuxFCA.System,
				FromStore: linuxFCA.FromStore,
				FromPath:  linuxFCA.FromPath,
				ToPath:    linuxFCA.ToPath,
			}
		}
	}
	return pkg
}
