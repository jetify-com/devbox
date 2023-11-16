//nolint:dupl // add/exclude platforms needs refactoring
package devconfig

import (
	"encoding/json"
	"io"
	"slices"
	"strings"

	"github.com/pkg/errors"
	orderedmap "github.com/wk8/go-ordered-map/v2"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/ux"
)

type Packages struct {
	// Collection contains the set of package definitions
	Collection []Package

	ast *configAST
}

// VersionedNames returns a list of package names with versions.
// NOTE: if the package is unversioned, the version will be omitted (doesn't default to @latest).
//
// example:
// ["package1", "package2@latest", "package3@1.20"]
func (pkgs *Packages) VersionedNames() []string {
	result := make([]string, 0, len(pkgs.Collection))
	for _, p := range pkgs.Collection {
		result = append(result, p.VersionedName())
	}
	return result
}

// Get returns the package with the given versionedName
func (pkgs *Packages) Get(versionedName string) (*Package, bool) {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return nil, false
	}
	return &pkgs.Collection[i], true
}

// Add adds a package to the list of packages
func (pkgs *Packages) Add(versionedName string) {
	name, version := parseVersionedName(versionedName)
	if pkgs.index(name, version) != -1 {
		return
	}
	pkgs.Collection = append(pkgs.Collection, NewVersionOnlyPackage(name, version))
	pkgs.ast.appendPackage(name, version)
}

// Remove removes a package from the list of packages
func (pkgs *Packages) Remove(versionedName string) {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return
	}
	pkgs.Collection = slices.Delete(pkgs.Collection, i, i+1)
	pkgs.ast.removePackage(name)
}

// AddPlatforms adds a platform to the list of platforms for a given package
func (pkgs *Packages) AddPlatforms(writer io.Writer, versionedname string, platforms []string) error {
	if len(platforms) == 0 {
		return nil
	}
	if err := nix.EnsureValidPlatform(platforms...); err != nil {
		return errors.WithStack(err)
	}

	name, version := parseVersionedName(versionedname)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedname)
	}

	// Adding any platform will restrict installation to it, so
	// the ExcludedPlatforms are no longer needed
	pkg := &pkgs.Collection[i]
	if len(pkg.ExcludedPlatforms) > 0 {
		return usererr.New(
			"cannot add any platform for package %s because it already has `excluded_platforms` defined. "+
				"Please delete the `excluded_platforms` for this package from devbox.json and retry.",
			pkg.VersionedName(),
		)
	}

	// Append if the platform is not already present
	oldLen := len(pkg.Platforms)
	for _, p := range platforms {
		if !slices.Contains(pkg.Platforms, p) {
			pkg.Platforms = append(pkg.Platforms, p)
		}
	}
	if len(pkg.Platforms) > oldLen {
		pkgs.ast.appendPlatforms(pkg.name, "platforms", pkg.Platforms[oldLen:])
		ux.Finfo(writer,
			"Added platform %s to package %s\n", strings.Join(platforms, ", "),
			pkg.VersionedName(),
		)
	}
	return nil
}

// ExcludePlatforms adds a platform to the list of excluded platforms for a given package
func (pkgs *Packages) ExcludePlatforms(writer io.Writer, versionedName string, platforms []string) error {
	if len(platforms) == 0 {
		return nil
	}
	if err := nix.EnsureValidPlatform(platforms...); err != nil {
		return errors.WithStack(err)
	}

	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedName)
	}

	pkg := &pkgs.Collection[i]
	if len(pkg.Platforms) > 0 {
		return usererr.New(
			"cannot exclude any platform for package %s because it already has `platforms` defined. "+
				"Please delete the `platforms` for this package from devbox.json and re-try.",
			pkg.VersionedName(),
		)
	}

	oldLen := len(pkg.ExcludedPlatforms)
	for _, p := range platforms {
		if !slices.Contains(pkg.ExcludedPlatforms, p) {
			pkg.ExcludedPlatforms = append(pkg.ExcludedPlatforms, p)
		}
	}
	if len(pkg.ExcludedPlatforms) > oldLen {
		pkgs.ast.appendPlatforms(pkg.name, "excluded_platforms", pkg.ExcludedPlatforms[oldLen:])
		ux.Finfo(writer, "Excluded platform %s for package %s\n", strings.Join(platforms, ", "),
			pkg.VersionedName())
	}
	return nil
}

func (pkgs *Packages) UnmarshalJSON(data []byte) error {
	// First, attempt to unmarshal as a list of strings (legacy format)
	var packages []string
	if err := json.Unmarshal(data, &packages); err == nil {
		pkgs.Collection = packagesFromLegacyList(packages)
		return nil
	}

	// Second, attempt to unmarshal as a map of Packages
	// We use orderedmap to preserve the order of the packages. While the JSON
	// specification specifies that maps are unordered, we do rely on the order
	// for certain functionality.
	orderedMap := orderedmap.New[string, Package]()
	err := json.Unmarshal(data, &orderedMap)
	if err != nil {
		return errors.WithStack(err)
	}

	// Convert the ordered map to a list of packages, and set the name field
	// from the map's key
	packagesList := []Package{}
	for pair := orderedMap.Oldest(); pair != nil; pair = pair.Next() {
		pkg := pair.Value
		pkg.name = pair.Key
		packagesList = append(packagesList, pkg)
	}
	pkgs.Collection = packagesList
	return nil
}

func (pkgs *Packages) index(name, version string) int {
	return slices.IndexFunc(pkgs.Collection, func(p Package) bool {
		return p.name == name && p.Version == version
	})
}

type Package struct {
	name    string
	Version string `json:"version,omitempty"`

	Platforms         []string `json:"platforms,omitempty"`
	ExcludedPlatforms []string `json:"excluded_platforms,omitempty"`

	// PatchGlibc applies a function to the package's derivation that
	// patches any ELF binaries to use the latest version of nixpkgs#glibc.
	PatchGlibc bool `json:"patch_glibc,omitempty"`
}

func NewVersionOnlyPackage(name, version string) Package {
	return Package{
		name:    name,
		Version: version,
	}
}

func NewPackage(name string, values map[string]any) Package {
	version, ok := values["version"]
	if !ok {
		// For legacy packages, the version may not be specified. We leave it blank
		// here, and code that consumes the Config is expected to handle this case
		// (e.g. by defaulting to @latest).
		version = ""
	}

	var platforms []string
	if p, ok := values["platforms"]; ok {
		platforms = p.([]string)
	}
	var excludedPlatforms []string
	if e, ok := values["excluded_platforms"]; ok {
		excludedPlatforms = e.([]string)
	}

	return Package{
		name:              name,
		Version:           version.(string),
		Platforms:         platforms,
		ExcludedPlatforms: excludedPlatforms,
	}
}

// enabledOnPlatform returns whether the package is enabled on the given platform.
// If the package has a list of platforms, it is enabled only on those platforms.
// If the package has a list of excluded platforms, it is enabled on all platforms
// except those.
func (p *Package) IsEnabledOnPlatform() bool {
	platform := nix.System()
	if len(p.Platforms) > 0 {
		for _, plt := range p.Platforms {
			if plt == platform {
				return true
			}
		}
		return false
	}
	for _, plt := range p.ExcludedPlatforms {
		if plt == platform {
			return false
		}
	}
	return true
}

func (p *Package) VersionedName() string {
	name := p.name
	if p.Version != "" {
		name += "@" + p.Version
	}
	return name
}

func (p *Package) UnmarshalJSON(data []byte) error {
	// First, attempt to unmarshal as a version-only string
	var version string
	if err := json.Unmarshal(data, &version); err == nil {
		p.Version = version
		return nil
	}

	// Second, attempt to unmarshal as a Package struct
	type packageAlias Package // Use an alias-type to avoid infinite recursion
	alias := &packageAlias{}
	if err := json.Unmarshal(data, alias); err != nil {
		return errors.WithStack(err)
	}

	*p = Package(*alias)
	return nil
}

// parseVersionedName parses the name and version from package@version representation
func parseVersionedName(versionedName string) (name, version string) {
	var found bool
	name, version, found = searcher.ParseVersionedPackage(versionedName)
	if !found {
		// Case without any @version in the versionedName
		// We deliberately do not set version to `latest`
		return versionedName, "" /*version*/
	}
	return name, version
}

// packagesFromLegacyList converts a list of strings to a list of packages
// Example inputs: `["python@latest", "hello", "cowsay@1"]`
func packagesFromLegacyList(packages []string) []Package {
	packagesList := []Package{}
	for _, p := range packages {
		name, version := parseVersionedName(p)
		packagesList = append(packagesList, NewVersionOnlyPackage(name, version))
	}
	return packagesList
}
