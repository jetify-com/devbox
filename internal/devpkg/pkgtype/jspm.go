package pkgtype

import "strings"

// JSPackageManager represents which JS package manager manages a package.
type JSPackageManager string

const (
	Pnpm JSPackageManager = "pnpm"
	Yarn JSPackageManager = "yarn"
	Npm  JSPackageManager = "npm"

	PnpmScheme = "pnpm"
	PnpmPrefix = PnpmScheme + ":"
	YarnScheme = "yarn"
	YarnPrefix = YarnScheme + ":"
	NpmScheme  = "npm"
	NpmPrefix  = NpmScheme + ":"
)

// IsJSPM returns true if the string has a pnpm:, yarn:, or npm: prefix.
func IsJSPM(s string) bool {
	return strings.HasPrefix(s, PnpmPrefix) ||
		strings.HasPrefix(s, YarnPrefix) ||
		strings.HasPrefix(s, NpmPrefix)
}

// JSPMType returns which JS package manager is indicated by the prefix.
// Panics if the string is not a JSPM package.
func JSPMType(s string) JSPackageManager {
	switch {
	case strings.HasPrefix(s, PnpmPrefix):
		return Pnpm
	case strings.HasPrefix(s, YarnPrefix):
		return Yarn
	case strings.HasPrefix(s, NpmPrefix):
		return Npm
	default:
		panic("not a JSPM package: " + s)
	}
}

// JSPMPackageName strips the prefix and splits on @ to return (name, version).
// For "pnpm:vercel@latest" returns ("vercel", "latest").
// For "pnpm:vercel" returns ("vercel", "").
func JSPMPackageName(raw string) (name, version string) {
	// Strip the prefix
	pkg := raw
	switch {
	case strings.HasPrefix(pkg, PnpmPrefix):
		pkg = strings.TrimPrefix(pkg, PnpmPrefix)
	case strings.HasPrefix(pkg, YarnPrefix):
		pkg = strings.TrimPrefix(pkg, YarnPrefix)
	case strings.HasPrefix(pkg, NpmPrefix):
		pkg = strings.TrimPrefix(pkg, NpmPrefix)
	}

	// Split on last @ (to handle scoped packages like @scope/pkg@version)
	if i := strings.LastIndex(pkg, "@"); i > 0 {
		return pkg[:i], pkg[i+1:]
	}
	return pkg, ""
}
