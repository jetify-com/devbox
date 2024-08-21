//nolint:dupl // add/exclude platforms needs refactoring
package configfile

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

type PackagesMutator struct {
	// collection contains the set of package definitions
	collection []Package

	ast *configAST
}

// Add adds a package to the list of packages
func (pkgs *PackagesMutator) Add(versionedName string) {
	name, version := parseVersionedName(versionedName)
	if pkgs.index(name, version) != -1 {
		return
	}
	pkgs.collection = append(pkgs.collection, NewVersionOnlyPackage(name, version))
	pkgs.ast.appendPackage(name, version)
}

// Remove removes a package from the list of packages
func (pkgs *PackagesMutator) Remove(versionedName string) {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return
	}
	pkgs.collection = slices.Delete(pkgs.collection, i, i+1)
	pkgs.ast.removePackage(name)
}

// AddPlatforms adds a platform to the list of platforms for a given package
func (pkgs *PackagesMutator) AddPlatforms(writer io.Writer, versionedname string, platforms []string) error {
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
	pkg := &pkgs.collection[i]
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
		pkgs.ast.appendPlatforms(pkg.Name, "platforms", pkg.Platforms[oldLen:])
		ux.Finfof(writer,
			"Added platform %s to package %s\n", strings.Join(platforms, ", "),
			pkg.VersionedName(),
		)
	}
	return nil
}

// ExcludePlatforms adds a platform to the list of excluded platforms for a given package
func (pkgs *PackagesMutator) ExcludePlatforms(writer io.Writer, versionedName string, platforms []string) error {
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

	pkg := &pkgs.collection[i]
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
		pkgs.ast.appendPlatforms(pkg.Name, "excluded_platforms", pkg.ExcludedPlatforms[oldLen:])
		ux.Finfof(writer, "Excluded platform %s for package %s\n", strings.Join(platforms, ", "),
			pkg.VersionedName())
	}
	return nil
}

func (pkgs *PackagesMutator) UnmarshalJSON(data []byte) error {
	// First, attempt to unmarshal as a list of strings (legacy format)
	var packages []string
	if err := json.Unmarshal(data, &packages); err == nil {
		pkgs.collection = packagesFromLegacyList(packages)
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
		pkg.Name = pair.Key
		packagesList = append(packagesList, pkg)
	}
	pkgs.collection = packagesList
	return nil
}

func (pkgs *PackagesMutator) SetPatchGLibc(versionedName string, v bool) error {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedName)
	}
	if pkgs.collection[i].PatchGlibc != v {
		pkgs.collection[i].PatchGlibc = v
		pkgs.ast.setPackageBool(name, "patch_glibc", v)
	}
	return nil
}

func (pkgs *PackagesMutator) SetDisablePlugin(versionedName string, v bool) error {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedName)
	}
	if pkgs.collection[i].DisablePlugin != v {
		pkgs.collection[i].DisablePlugin = v
		pkgs.ast.setPackageBool(name, "disable_plugin", v)
	}
	return nil
}

func (pkgs *PackagesMutator) SetOutputs(writer io.Writer, versionedName string, outputs []string) error {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedName)
	}

	toAdd := []string{}
	for _, o := range outputs {
		if !slices.Contains(pkgs.collection[i].Outputs, o) {
			toAdd = append(toAdd, o)
		}
	}

	if len(toAdd) > 0 {
		pkg := &pkgs.collection[i]
		pkgs.ast.appendOutputs(pkg.Name, "outputs", toAdd)
		ux.Finfof(writer, "Added outputs %s to package %s\n", strings.Join(toAdd, ", "), versionedName)
	}
	return nil
}

func (pkgs *PackagesMutator) SetAllowInsecure(writer io.Writer, versionedName string, whitelist []string) error {
	name, version := parseVersionedName(versionedName)
	i := pkgs.index(name, version)
	if i == -1 {
		return errors.Errorf("package %s not found", versionedName)
	}

	toAdd := []string{}
	for _, w := range whitelist {
		if !slices.Contains(pkgs.collection[i].AllowInsecure, w) {
			toAdd = append(toAdd, w)
		}
	}

	if len(toAdd) > 0 {
		pkg := &pkgs.collection[i]
		pkgs.ast.appendAllowInsecure(pkg.Name, "allow_insecure", toAdd)
		pkg.AllowInsecure = append(pkg.AllowInsecure, toAdd...)
		ux.Finfof(writer, "Allowed insecure %s for package %s\n", strings.Join(toAdd, ", "), versionedName)
	}
	return nil
}

func (pkgs *PackagesMutator) index(name, version string) int {
	return slices.IndexFunc(pkgs.collection, func(p Package) bool {
		return p.Name == name && p.Version == version
	})
}

type Package struct {
	Name    string
	Version string `json:"version,omitempty"`

	DisablePlugin     bool     `json:"disable_plugin,omitempty"`
	Platforms         []string `json:"platforms,omitempty"`
	ExcludedPlatforms []string `json:"excluded_platforms,omitempty"`

	// PatchGlibc applies a function to the package's derivation that
	// patches any ELF binaries to use the latest version of nixpkgs#glibc.
	PatchGlibc bool `json:"patch_glibc,omitempty"`

	// Outputs is the list of outputs to use for this package, assuming
	// it is a nix package. If empty, the default output is used.
	Outputs []string `json:"outputs,omitempty"`

	// AllowInsecure is a whitelist of packages that may be marked insecure
	// in nixpkgs, but are allowed by the user to be installed.
	AllowInsecure []string `json:"allow_insecure,omitempty"`
}

func NewVersionOnlyPackage(name, version string) Package {
	return Package{
		Name:    name,
		Version: version,
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
	name := p.Name
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
