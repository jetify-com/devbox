// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"fmt"
	"strings"
)

// Ref is a parsed "pkg:<source>@<version>" package string.
type Ref struct {
	// Source locates the definition file: a path relative to the project
	// directory, an absolute path, or an https:// or file:// URL.
	Source string

	// Version is the requested version constraint ("latest", "1.22",
	// "1.22.3"). It is empty when the package string has no "@version".
	Version string
}

// IsRef reports whether s is a package definition reference, i.e. it has
// the "pkg:" scheme.
func IsRef(s string) bool {
	return strings.HasPrefix(s, Prefix)
}

// ParseRef parses a "pkg:<source>@<version>" string. The version is split
// at the last "@" so that sources may themselves contain "@".
func ParseRef(raw string) (Ref, error) {
	if !IsRef(raw) {
		return Ref{}, fmt.Errorf("package %q: missing %q scheme", raw, Prefix)
	}
	rest := strings.TrimPrefix(raw, Prefix)
	ref := Ref{Source: rest}
	if i := strings.LastIndex(rest, "@"); i >= 0 {
		ref.Source = rest[:i]
		ref.Version = rest[i+1:]
	}
	if ref.Source == "" {
		return Ref{}, fmt.Errorf("package %q: missing definition path or URL", raw)
	}
	if !ref.IsRemote() && !strings.HasPrefix(ref.Source, "file://") {
		// Guard against schemes we don't support (github:, ssh:, ...) being
		// mistaken for relative paths. A path segment before the first
		// colon contains "/" or ".", a scheme does not.
		if scheme, _, ok := strings.Cut(ref.Source, ":"); ok && !strings.ContainsAny(scheme, "/.") {
			return Ref{}, fmt.Errorf("package %q: unsupported definition source scheme %q", raw, scheme)
		}
	}
	return ref, nil
}

// String formats the reference back into its devbox.json form.
func (r Ref) String() string {
	s := Prefix + r.Source
	if r.Version != "" {
		s += "@" + r.Version
	}
	return s
}

// WithVersion returns a copy of the reference with a different version.
func (r Ref) WithVersion(version string) Ref {
	r.Version = version
	return r
}

// IsRemote reports whether the source is fetched over HTTP(S).
func (r Ref) IsRemote() bool {
	return strings.HasPrefix(r.Source, "https://") || strings.HasPrefix(r.Source, "http://")
}

// IsLocal reports whether the source is read from the local filesystem,
// either as a path or as a file:// URL.
func (r Ref) IsLocal() bool {
	if strings.HasPrefix(r.Source, "file://") {
		return true
	}
	return !strings.Contains(r.Source, "://")
}
