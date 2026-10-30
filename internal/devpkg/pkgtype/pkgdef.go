package pkgtype

import (
	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/pkgdef"
)

// IsPkgDef reports whether s is a package definition reference
// ("pkg:<ref>@<version>", see specs/package-definitions.md). It is always
// false while the PKG_DEFS feature flag is off, in which case such strings
// fall through to the regular (and failing) nixpkgs lookup.
func IsPkgDef(s string) bool {
	return featureflag.PkgDefs.Enabled() && pkgdef.IsRef(s)
}
