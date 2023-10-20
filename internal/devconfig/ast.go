package devconfig

import (
	"bytes"
	"slices"

	"github.com/tailscale/hujson"
)

// configAST is a hujson syntax tree that represents a devbox.json
// configuration. An AST allows the CLI to modify specific parts of a user's
// devbox.json instead of overwriting the entire file. This is important
// because a devbox.json can have user comments that must be preserved when
// saving changes.
//
//   - Unmarshalling is still done with encoding/json.
//   - Marshalling is done by calling configAST.root.Pack to encode the AST as
//     hujson/JWCC. Therefore, any changes to a Config struct will NOT
//     automatically be marshaled back to JSON. Support for modifying a part of
//     the JSON must be explicitly implemented in configAST.
//   - Validation with the AST is complex, so it doesn't do any. It will happily
//     append duplicate object keys and panic on invalid types. The higher-level
//     Config type is responsible for tracking state and making valid edits to
//     the AST.
//
// Be aware that there are 4 ways of representing a package in devbox.json that
// the AST needs to handle:
//
//  1. ["name"] or ["name@version"] (versioned name array)
//  2. {"name": "version"} (packages object member with version string)
//  3. {"name": {"version": "1.2.3"}} (packages object member with package object)
//  4. {"github:F1bonacc1/process-compose/v0.40.2": {}} (packages object member with flakeref)
type configAST struct {
	root hujson.Value
}

// parseConfig parses the bytes of a devbox.json and returns a syntax tree.
func parseConfig(b []byte) (*configAST, error) {
	root, err := hujson.Parse(b)
	if err != nil {
		return nil, err
	}
	return &configAST{root: root}, nil
}

// packagesField gets the "packages" field, initializing it if necessary. The
// member value will either be an array of strings or an object. When it's an
// object, the keys will always be package names and the values will be a
// string or another object. Examples are:
//
//   - {"packages": ["go", "hello"]}
//   - {"packages": {"go": "1.20", "hello: {"platforms": ["aarch64-darwin"]}}}
//
// When migrate is true, the packages value will be migrated from the legacy
// array format to the object format. For example, the array:
//
//	["go@latest", "hello"]
//
// will become:
//
//	{
//		"go": "latest",
//		"hello": ""
//	}
func (c *configAST) packagesField(migrate bool) *hujson.ObjectMember {
	rootObject := c.root.Value.(*hujson.Object)
	i := c.memberIndex(rootObject, "packages")
	if i != -1 {
		switch rootObject.Members[i].Value.Value.Kind() {
		case '[':
			if migrate {
				c.migratePackagesArray(&rootObject.Members[i].Value)
				c.root.Format()
			}
		case 'n':
			// Initialize a null packages field to an empty object.
			rootObject.Members[i].Value.Value = &hujson.Object{
				AfterExtra: []byte{'\n'},
			}
			c.root.Format()
		}
		return &rootObject.Members[i]
	}

	// Add a packages field to the root config object and initialize it with
	// an empty object.
	rootObject.Members = append(rootObject.Members, hujson.ObjectMember{
		Name: hujson.Value{
			Value:       hujson.String("packages"),
			BeforeExtra: []byte{'\n'},
		},
		Value: hujson.Value{Value: &hujson.Object{}},
	})
	c.root.Format()
	return &rootObject.Members[len(rootObject.Members)-1]
}

// appendPackage appends a package to the packages field.
func (c *configAST) appendPackage(name, version string) {
	pkgs := c.packagesField(false)
	switch val := pkgs.Value.Value.(type) {
	case *hujson.Object:
		c.appendPackageToObject(val, name, version)
	case *hujson.Array:
		c.appendPackageToArray(val, joinNameVersion(name, version))
	default:
		panic("packages field must be an object or array")
	}

	// Ensure the packages field is on its own line.
	if !slices.Contains(pkgs.Name.BeforeExtra, '\n') {
		pkgs.Name.BeforeExtra = append(pkgs.Name.BeforeExtra, '\n')
	}
	c.root.Format()
}

func (c *configAST) appendPackageToObject(pkgs *hujson.Object, name, version string) {
	i := c.memberIndex(pkgs, name)
	if i != -1 {
		return
	}

	// Add a new member to the packages object with the package name and
	// version.
	pkgs.Members = append(pkgs.Members, hujson.ObjectMember{
		Name:  hujson.Value{Value: hujson.String(name), BeforeExtra: []byte{'\n'}},
		Value: hujson.Value{Value: hujson.String(version)},
	})
}

func (c *configAST) appendPackageToArray(arr *hujson.Array, versionedName string) {
	var extra []byte
	if len(arr.Elements) > 0 {
		// Put each element on its own line if there
		// will be more than 1.
		extra = []byte{'\n'}
	}
	arr.Elements = append(arr.Elements, hujson.Value{
		BeforeExtra: extra,
		Value:       hujson.String(versionedName),
	})
}

// removePackage removes a package from the packages field.
func (c *configAST) removePackage(name string) {
	switch val := c.packagesField(false).Value.Value.(type) {
	case *hujson.Object:
		c.removePackageMember(val, name)
	case *hujson.Array:
		c.removePackageElement(val, name)
	default:
		panic("packages field must be an object or array")
	}
	c.root.Format()
}

func (c *configAST) removePackageMember(pkgs *hujson.Object, name string) {
	i := c.memberIndex(pkgs, name)
	if i == -1 {
		return
	}
	pkgs.Members = slices.Delete(pkgs.Members, i, i+1)
}

func (c *configAST) removePackageElement(arr *hujson.Array, name string) {
	i := c.packageElementIndex(arr, name)
	if i == -1 {
		return
	}
	arr.Elements = slices.Delete(arr.Elements, i, i+1)
}

// appendPlatforms appends a platform to a package's "platforms" or
// "excluded_platforms" field. It automatically converts the package to an
// object if it isn't already.
func (c *configAST) appendPlatforms(name, fieldName string, platforms []string) {
	if len(platforms) == 0 {
		return
	}

	pkgs := c.packagesField(true).Value.Value.(*hujson.Object)
	i := c.memberIndex(pkgs, name)
	if i == -1 {
		return
	}

	// We need to ensure that the package value is a full object
	// (not a version string) before we can add a platform.
	c.convertVersionToObject(&pkgs.Members[i].Value)

	pkgObject := pkgs.Members[i].Value.Value.(*hujson.Object)
	var arr *hujson.Array
	if i := c.memberIndex(pkgObject, fieldName); i == -1 {
		arr = &hujson.Array{
			Elements: make([]hujson.Value, 0, len(platforms)),
		}
		pkgObject.Members = append(pkgObject.Members, hujson.ObjectMember{
			Name: hujson.Value{
				Value:       hujson.String(fieldName),
				BeforeExtra: []byte{'\n'},
			},
			Value: hujson.Value{Value: arr},
		})
	} else {
		arr = pkgObject.Members[i].Value.Value.(*hujson.Array)
		arr.Elements = slices.Grow(arr.Elements, len(platforms))
	}

	for _, p := range platforms {
		arr.Elements = append(arr.Elements, hujson.Value{Value: hujson.String(p)})
	}
	c.root.Format()
}

// migratePackagesArray migrates a legacy array of package versionedNames to an
// object. See packagesField for details.
func (c *configAST) migratePackagesArray(pkgs *hujson.Value) {
	arr := pkgs.Value.(*hujson.Array)
	obj := &hujson.Object{Members: make([]hujson.ObjectMember, len(arr.Elements))}
	for i, elem := range arr.Elements {
		name, version := parseVersionedName(elem.Value.(hujson.Literal).String())

		// Preserve any comments above the array elements.
		var before []byte
		if comment := bytes.TrimSpace(elem.BeforeExtra); len(comment) > 0 {
			before = append([]byte{'\n'}, comment...)
		}
		before = append(before, '\n')

		obj.Members[i] = hujson.ObjectMember{
			Name: hujson.Value{
				Value:       hujson.String(name),
				BeforeExtra: before,
			},
			Value: hujson.Value{Value: hujson.String(version)},
		}
	}
	pkgs.Value = obj
}

// convertVersionToObject transforms a version string into an object with the
// version as a field.
func (c *configAST) convertVersionToObject(pkg *hujson.Value) {
	if pkg.Value.Kind() == '{' {
		return
	}

	obj := &hujson.Object{}
	if version, ok := pkg.Value.(hujson.Literal); ok && version.String() != "" {
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name: hujson.Value{
				Value:       hujson.String("version"),
				BeforeExtra: []byte{'\n'},
			},
			Value: hujson.Value{Value: version},
		})
	}
	pkg.Value = obj
}

// memberIndex returns the index of an object member.
func (*configAST) memberIndex(obj *hujson.Object, name string) int {
	return slices.IndexFunc(obj.Members, func(m hujson.ObjectMember) bool {
		return m.Name.Value.(hujson.Literal).String() == name
	})
}

// packageElementIndex returns the index of a package from an array of
// versionedName strings.
func (*configAST) packageElementIndex(arr *hujson.Array, name string) int {
	return slices.IndexFunc(arr.Elements, func(v hujson.Value) bool {
		elemName, _ := parseVersionedName(v.Value.(hujson.Literal).String())
		return elemName == name
	})
}

func joinNameVersion(name, version string) string {
	if version == "" {
		return name
	}
	return name + "@" + version
}
