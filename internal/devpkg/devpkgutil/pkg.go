// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkgutil

import (
	"regexp"
	"strings"
)

// ParseVersionedPackage checks if the given package is a versioned package (`python@3.10`)
// and returns its name and version
func ParseVersionedPackage(pkg string) (string, string, bool) {
	name, version, found := strings.Cut(pkg, "@")
	return name, version, found && name != "" && version != ""
}

// IsGithubNixpkgsURL returns true if the package is a flake of the form:
// github:NixOS/nixpkgs/...
//
// While there are many ways to specify this input, devbox always uses
// github:NixOS/nixpkgs/<hash> as the URL. If the user wishes to reference nixpkgs
// themselves, this function may not return true.
func IsGithubNixpkgsURL(url string) bool {
	return strings.HasPrefix(url, "github:NixOS/nixpkgs/")
}

var hashFromNixPkgsRegex = regexp.MustCompile(`github:NixOS/nixpkgs/([^#]+).*`)

// HashFromNixPkgsURL will (for example) return 5233fd2ba76a3accb5aaa999c00509a11fd0793c
// from github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello
func HashFromNixPkgsURL(url string) string {
	matches := hashFromNixPkgsRegex.FindStringSubmatch(url)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
