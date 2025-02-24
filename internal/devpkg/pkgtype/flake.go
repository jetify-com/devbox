package pkgtype

import (
	"strings"

	"go.jetify.com/devbox/nix/flake"
)

func IsFlake(s string) bool {
	if IsRunX(s) {
		return false
	}
	parsed, err := flake.ParseInstallable(s)
	if err != nil {
		return false
	}
	if IsAmbiguous(s, parsed) {
		return false
	}
	return true
}

// IsAmbiguous returns true if a package string could be a Devbox package or
// a flake installable. For example, "nixpkgs" is both a Devbox package and a
// flake.
func IsAmbiguous(raw string, parsed flake.Installable) bool {
	// Devbox package strings never have a #attr_path in them.
	if parsed.AttrPath != "" {
		return false
	}

	// Indirect installables must have a "flake:" scheme to disambiguate
	// them from legacy (unversioned) devbox package strings.
	if parsed.Ref.Type == flake.TypeIndirect {
		return !strings.HasPrefix(raw, "flake:")
	}

	// Path installables must have a "path:" scheme, start with "/" or start
	// with "./" to disambiguate them from devbox package strings.
	if parsed.Ref.Type == flake.TypePath {
		if raw[0] == '.' || raw[0] == '/' {
			return false
		}
		if strings.HasPrefix(raw, "path:") {
			return false
		}
		return true
	}

	// All other flakeref types must have a scheme, so we know those can't
	// be devbox package strings.
	return false
}
