package devconfig

import (
	"encoding/json"
	"strings"

	"github.com/pkg/errors"
	orderedmap "github.com/wk8/go-ordered-map/v2"
)

type jsonKind int

const (
	// jsonList is the legacy format for packages
	jsonList jsonKind = iota
	// jsonMap is the new format for packages
	jsonMap jsonKind = iota
)

type Packages struct {
	jsonKind jsonKind

	// Collection contains the set of package definitions
	// We don't want this key to be serialized automatically, hence the "key" in json is "-"
	// NOTE: this is not a pointer to make debugging failure cases easier
	// (get dumps of the values, not memory addresses)
	Collection []Package `json:"-,omitempty"`
}

// VersionedNames returns a list of package names with versions.
// NOTE: if the package is unversioned, the version will be omitted (doesn't default to @latest).
//
// example:
// ["package1", "package2@latest", "package3@1.20"]
func (pkgs *Packages) VersionedNames() []string {
	result := []string{}
	for _, p := range pkgs.Collection {
		name := p.name
		if p.Version != "" {
			name += "@" + p.Version
		}
		result = append(result, name)
	}
	return result
}

// Add adds a package to the list of packages
func (pkgs *Packages) Add(versionedName string) {
	name, version := parseVersionedName(versionedName)
	pkgs.Collection = append(pkgs.Collection, NewVersionOnlyPackage(name, version))
}

// Remove removes a package from the list of packages
func (pkgs *Packages) Remove(versionedName string) error {
	name, version := parseVersionedName(versionedName)
	for idx, pkg := range pkgs.Collection {
		if pkg.name == name && pkg.Version == version {
			pkgs.Collection = append(pkgs.Collection[:idx], pkgs.Collection[idx+1:]...)
			return nil
		}
	}
	return errors.Errorf("package %s not found", versionedName)
}

func (pkgs *Packages) UnmarshalJSON(data []byte) error {

	// First, attempt to unmarshal as a list of strings (legacy format)
	var packages []string
	if err := json.Unmarshal(data, &packages); err == nil {
		pkgs.Collection = packagesFromLegacyList(packages)
		pkgs.jsonKind = jsonList
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
	pkgs.jsonKind = jsonMap
	return nil
}

func (pkgs *Packages) MarshalJSON() ([]byte, error) {
	if pkgs.jsonKind == jsonList {
		packagesList := []string{}
		for _, p := range pkgs.Collection {

			// Version may be empty for unversioned packages
			packageToWrite := p.name
			if p.Version != "" {
				packageToWrite += "@" + p.Version
			}
			packagesList = append(packagesList, packageToWrite)
		}
		return json.Marshal(packagesList)
	}

	orderedMap := orderedmap.New[string, Package]()
	for _, p := range pkgs.Collection {
		orderedMap.Set(p.name, p)
	}
	return json.Marshal(orderedMap)
}

type packageKind int

const (
	versionOnly packageKind = iota
	regular     packageKind = iota
)

type Package struct {
	kind packageKind
	name string
	// deliberately not adding omitempty
	Version string `json:"version"`

	// TODO: add other fields like platforms
}

func NewVersionOnlyPackage(name, version string) Package {
	return Package{
		kind:    versionOnly,
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

	return Package{
		kind:    regular,
		name:    name,
		Version: version.(string),
	}
}

func (p *Package) UnmarshalJSON(data []byte) error {
	// First, attempt to unmarshal as a version-only string
	var version string
	if err := json.Unmarshal(data, &version); err == nil {
		p.kind = versionOnly
		p.Version = version
		return nil
	}

	// Second, attempt to unmarshal as a Package struct
	type Alias Package // Use an alias-type to avoid infinite recursion
	alias := &Alias{}
	if err := json.Unmarshal(data, alias); err != nil {
		return errors.WithStack(err)
	}

	// more robust way to copy all fields from alias?
	p.kind = regular
	p.Version = alias.Version
	return nil
}

func (p Package) MarshalJSON() ([]byte, error) {
	if p.kind == versionOnly {
		return json.Marshal(p.Version)
	}

	// If we have a regular package, we want to marshal the entire struct:
	type Alias Package // Use an alias-type to avoid infinite recursion
	return json.Marshal((Alias)(p))
}

// parseVersionedName parses the name and version from package@version representation
func parseVersionedName(versionedName string) (name, version string) {
	// use the last @ symbol as the version delimiter, some packages have @ in the name
	atSymbolIndex := strings.LastIndex(versionedName, "@")
	if atSymbolIndex != -1 {
		// Common case: package@version
		if atSymbolIndex != len(versionedName)-1 {
			name, version = versionedName[:atSymbolIndex], versionedName[atSymbolIndex+1:]
		} else {
			// This case handles packages that end with `@` in the name
			// example: `emacsPackages.@`
			name = versionedName[:atSymbolIndex] + "@"
		}
	} else {
		// Case without any @version: package
		name = versionedName

		// We deliberately do not set version to latest so that we don't
		// automatically modify the devbox.json file. It should only be modified
		// upon `devbox update`.
		// version = "latest"
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
