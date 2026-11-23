package pkgtype

import (
	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/pkgdef"
)

// IsPkgDef reports whether s is a package definition reference
// ("pkg:<ref>@<version>", see specs/package-definitions.md). It is always
// false while the PKG_DEFS feature flag is off, in which case such strings
// fall through to the regular (and failing) nixpkgs lookup.
func IsPkgDef(s string) bool {
	return featureflag.PkgDefs.Enabled() && pkgdef.IsRef(s)
}

// CheckPkgDefEnabled returns an error when s is a package definition
// reference but the PKG_DEFS feature flag is off. Without the flag a "pkg:"
// string would otherwise be treated as a nixpkgs attribute or a flake
// reference and fail much later with a confusing error (or after a slow
// nixpkgs search).
func CheckPkgDefEnabled(s string) error {
	if !pkgdef.IsRef(s) || featureflag.PkgDefs.Enabled() {
		return nil
	}
	return usererr.New(
		"Package %q is a package definition, which is an experimental feature. "+
			"Set DEVBOX_FEATURE_PKG_DEFS=1 to enable it.", s,
	)
}
