package featureflag

// PkgDefs enables package definitions: installing packages from the
// vendor's binary distribution via "pkg:<ref>@<version>" package strings.
// See specs/package-definitions.md.
var PkgDefs = disable("PKG_DEFS").EnableOnDev()
