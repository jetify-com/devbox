package devpkg

import (
	"strings"

	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/pkg/runx/impl/types"
)

// PackageSpec specifies a Devbox package to install. Devbox supports a number
// of package spec syntaxes:
//
//	| Syntax           | Example                     | Description                          |
//	| ---------------- | --------------------------- | ------------------------------------ |
//	| name@version     | go@v1.22                    | Resolved w/ search service           |
//	| name             | go                          | Same as above; implies @latest       |
//	| flake            | github:/nixos/nixpkgs#go    | Matches Nix installable syntax       |
//	| attr path        | go                          | Equivalent to flake:nixpkgs#go       |
//	| legacy attr path | go                          | Uses deprecated nixpkgs.commit field |
//	| runx             | runx:golangci/golangci-lint | Experimental                         |
//
// Most package specs are ambiguous, making it impossible to tell which exact
// syntax the user intended. PackageSpec parses as many as it can and leaves it
// up to the caller to prioritize and resolve them.
//
// For example, the package spec "cachix" is a Devbox package (implied @latest),
// a flake ref (cachix), and an attribute path (nixpkgs#cachix). When resolving
// the package, Devbox must pick one (most likely cachix@latest).
type PackageSpec struct {
	// Name and Version are the parsed components of Devbox's name@version
	// package syntax.
	Name, Version string

	// Installable is the spec parsed as a Nix flake installable.
	Installable flake.Installable

	// AttrPathInstallable is the spec parsed as a nixpkgs attribute path.
	// If the project has a legacy nixpkgs commit field set, then
	// AttrPathInstallable will use it. Otherwise, it defaults to whatever
	// is in the user's flake registry, which is usually nixpkgs-unstable.
	AttrPathInstallable flake.Installable

	// RunX is the spec parsed as a RunX package reference.
	RunX types.PkgRef
}

// ParsePackageSpec parses a raw Devbox package specifier. nixpkgsCommit should
// be empty unless the Devbox project has the deprecated nixpkgs.commit field.
//
// Parsing is strictly syntactical. ParsePackageSpec does not make any network
// calls or execute any Nix commands to disambiguate the raw spec.
func ParsePackageSpec(raw, nixpkgsCommit string) PackageSpec {
	if raw == "" {
		return PackageSpec{}
	}
	if after, ok := strings.CutPrefix(raw, "runx:"); ok {
		runx, err := types.NewPkgRef(after)
		if err == nil {
			return PackageSpec{RunX: runx}
		}
	}

	spec := PackageSpec{}
	spec.Installable, _ = flake.ParseInstallable(raw)
	if spec.isInstallableUnambiguous(raw) {
		// Definitely a flake, no need to keep going.
		return spec
	}

	isValidAttrPath := !strings.ContainsRune(raw, '#')
	if !isValidAttrPath {
		// Not an attribute path, so can't be a Devbox package either.
		return spec
	}

	spec.AttrPathInstallable = flake.Installable{
		Ref: flake.Ref{
			Type: flake.TypeIndirect,
			ID:   "nixpkgs",
			Ref:  nixpkgsCommit,
		},
		AttrPath: raw,
	}
	if nixpkgsCommit != "" {
		// Don't interpret raw as a flake ref if its ambiguous.
		// Otherwise, we would end up with an Installable that doesn't
		// respect the nixpkgs.commit field.
		//
		// For example, "cachix" is an indirect flake reference that
		// installs the default package from the cachix flake. But when
		// nixpkgs.commit is set, the user is actually trying to install
		// nixpkgs/<ref>#cachix.
		spec.Installable = flake.Installable{}
	}

	i := strings.LastIndexByte(raw, '@')
	if i <= 0 || i == len(raw)-1 {
		// When a Devbox spec doesn't specify a version, we need
		// to check for the deprecated nixpkgs.commit field. If
		// it's set, then we treat the spec as an attribute
		// path. Otherwise, we assume the latest version.In
		// other words:
		//
		// 	{"packages":["go"]} -> go@latest
		// 	{"packages":["go"],"nixpkgs":{"commit":"abc"}} -> nixpkgs/abc#go
		if nixpkgsCommit == "" {
			spec.Name = raw
			spec.Version = "latest"
		}
		// Leave Name and Version empty; rely on
		// AttrPathInstallable for legacy-style packages.
		return spec
	}
	spec.Name, spec.Version = raw[:i], raw[i+1:]
	return spec
}

// isInstallableUnambiguous returns true if the raw, unparsed form of
// p.Installable has an explicit scheme or starts with "./" or "/". Unambiguous
// installables are never parsed as Devbox package names or attribute paths.
func (p PackageSpec) isInstallableUnambiguous(raw string) bool {
	// The scheme is optional for indirect and path flake types, so we need
	// to check explicitly.
	var (
		isFlake    = p.Installable.Ref.Type != ""
		isIndirect = p.Installable.Ref.Type == flake.TypeIndirect
		isPath     = p.Installable.Ref.Type == flake.TypePath
	)
	if isFlake && !isIndirect && !isPath {
		return true
	}
	if isIndirect && strings.HasPrefix(raw, "flake:") {
		return true
	}
	if isPath && strings.HasPrefix(raw, "path:") {
		return true
	}
	if isPath && (strings.HasPrefix(raw, "./") || strings.HasPrefix(raw, "/")) {
		return true
	}
	return false
}
