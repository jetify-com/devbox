// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/searcher"
)

type Input struct {
	url.URL
	lockfile lock.Locker
}

func InputFromString(s string, l lock.Locker) *Input {
	u, _ := url.Parse(s)
	if u.Path == "" && u.Opaque != "" && u.Scheme == "path" {
		// This normalizes url paths to be absolute. It also ensures all
		// path urls have a single slash (instead of possibly 3 slashes)
		u, _ = url.Parse("path:" + filepath.Join(l.ProjectDir(), u.Opaque))
	}
	return &Input{*u, l}
}

func (i *Input) IsLocal() bool {
	// Technically flakes allows omitting the scheme for local absolute paths, but
	// we don't support that (yet).
	return i.Scheme == "path"
}

func (i *Input) IsDevboxPackage() bool {
	return i.Scheme == ""
}

func (i *Input) IsGithub() bool {
	return i.Scheme == "github"
}

var inputNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]+")

func (i *Input) Name() string {
	result := ""
	if i.IsLocal() {
		result = filepath.Base(i.Path) + "-" + i.hash()
	} else if i.IsGithub() {
		result = "gh-" + strings.Join(strings.Split(i.Opaque, "/"), "-")
	} else if url := i.URLForInput(); IsGithubNixpkgsURL(url) {
		u := HashFromNixPkgsURL(url)
		if len(u) > 6 {
			u = u[0:6]
		}
		result = "nixpkgs-" + u
	} else {
		result = i.String() + "-" + i.hash()
	}
	return inputNameRegex.ReplaceAllString(result, "-")
}

func (i *Input) URLForInput() string {
	if i.IsDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			panic(err)
			// TODO(landau): handle error
		}
		withoutFragment, _, _ := strings.Cut(entry.Resolved, "#")
		return withoutFragment
	}
	return i.urlWithoutFragment()
}

func (i *Input) URLForInstall() (string, error) {
	if i.IsDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		return entry.Resolved, nil
	}
	attrPath, err := i.PackageAttributePath()
	if err != nil {
		return "", err
	}
	return i.urlWithoutFragment() + "#" + attrPath, nil
}

// PackageAttributePath returns just the name for non-flakes. For flake
// references is returns the full path to the package in the flake. e.g.
// packages.x86_64-linux.hello
func (i *Input) PackageAttributePath() (string, error) {
	var infos map[string]*Info
	if i.IsDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		infos = search(entry.Resolved)
	} else {
		infos = search(i.String())
	}

	if len(infos) == 1 {
		return lo.Keys(infos)[0], nil
	}

	// If ambiguous, try to find a default output
	if len(infos) > 1 && i.Fragment == "" {
		for key := range infos {
			if strings.HasSuffix(key, ".default") {
				return key, nil
			}
		}
		for key := range infos {
			if strings.HasPrefix(key, "defaultPackage.") {
				return key, nil
			}
		}
	}

	// Still ambiguous, return error
	if len(infos) > 1 {
		outputs := fmt.Sprintf("It has %d possible outputs", len(infos))
		if len(infos) < 10 {
			outputs = "It has the following possible outputs: \n" +
				strings.Join(lo.Keys(infos), ", ")
		}
		return "", usererr.New(
			"Package \"%s\" is ambiguous. %s",
			i.String(),
			outputs,
		)
	}

	return "", usererr.New("Package \"%s\" was not found", i.String())
}

func (i *Input) urlWithoutFragment() string {
	u := i.URL // get copy
	u.Fragment = ""
	return u.String()
}

func (i *Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) validateExists() (bool, error) {
	if i.isVersioned() {
		version := i.version()
		if version == "" && i.isVersioned() {
			return false, usererr.New("No version specified for %q.", i.Path)
		}
		return searcher.Exists(i.CanonicalName(), version)
	}
	info, err := i.PackageAttributePath()
	return info != "", err
}

func (i *Input) equals(other *Input) bool {
	if i.String() == other.String() {
		return true
	}

	// check inputs without fragments as optimization. Next step is expensive
	if i.URLForInput() != other.URLForInput() {
		return false
	}

	name, err := i.PackageAttributePath()
	if err != nil {
		return false
	}
	otherName, err := other.PackageAttributePath()
	if err != nil {
		return false
	}
	return name == otherName
}

// CanonicalName returns the name of the package without the version
// it only applies to devbox packages
func (i *Input) CanonicalName() string {
	if !i.IsDevboxPackage() {
		return ""
	}
	name, _, _ := strings.Cut(i.Path, "@")
	return name
}

// version returns the version of the package
// it only applies to devbox packages
func (i *Input) version() string {
	if !i.IsDevboxPackage() {
		return ""
	}
	_, version, _ := strings.Cut(i.Path, "@")
	return version
}

func (i *Input) isVersioned() bool {
	return i.IsDevboxPackage() && strings.Contains(i.Path, "@")
}

func (i *Input) hashFromNiPkgsURL() string {
	return HashFromNixPkgsURL(i.URLForInput())
}

// IsGithubNixpkgsURL returns true if the input is a nixpkgs flake of the form:
// github:NixOS/nixpkgs/...
//
// While there are many ways to specify this input, devbox always uses
// github:NixOS/nixpkgs/<hash> as the URL. If the user wishes to reference nixpkgs
// themselves, this function may not return true.
func IsGithubNixpkgsURL(url string) bool {
	return strings.HasPrefix(url, "github:NixOS/nixpkgs/")
}

func HashFromNixPkgsURL(url string) string {
	return strings.TrimPrefix(url, "github:NixOS/nixpkgs/")
}
