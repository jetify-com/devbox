// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/lock"
)

type Input struct {
	url.URL
	lockfile lock.Locker
	Raw      string
}

func InputsFromStrings(names []string, l lock.Locker) []*Input {
	inputs := []*Input{}
	for _, name := range names {
		inputs = append(inputs, InputFromString(name, l))
	}
	return inputs
}

func InputFromString(s string, l lock.Locker) *Input {
	u, _ := url.Parse(s)
	if u.Path == "" && u.Opaque != "" && u.Scheme == "path" {
		// This normalizes url paths to be absolute. It also ensures all
		// path urls have a single slash (instead of possibly 3 slashes)
		u, _ = url.Parse("path:" + filepath.Join(l.ProjectDir(), u.Opaque))
	}
	return &Input{*u, l, s}
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

func (i *Input) InputName() string {
	result := ""
	if i.IsLocal() {
		result = filepath.Base(i.Path) + "-" + i.Hash()
	} else if i.IsGithub() {
		result = "gh-" + strings.Join(strings.Split(i.Opaque, "/"), "-")
	} else if url := i.URLForInput(); IsGithubNixpkgsURL(url) {
		u := HashFromNixPkgsURL(url)
		if len(u) > 6 {
			u = u[0:6]
		}
		result = "nixpkgs-" + u
	} else {
		result = i.String() + "-" + i.Hash()
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
	attrPath, err := i.FullPackageAttributePath()
	if err != nil {
		return "", err
	}
	return i.urlWithoutFragment() + "#" + attrPath, nil
}

func (i *Input) normalizedDevboxPackageReference() (string, error) {
	if !i.IsDevboxPackage() {
		return "", nil
	}

	path := ""
	if i.isVersioned() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		path = entry.Resolved
	} else if i.IsDevboxPackage() {
		path = i.lockfile.LegacyNixpkgsPath(i.String())
	}

	if path != "" {
		s, err := System()
		if err != nil {
			return "", err
		}
		url, fragment, _ := strings.Cut(path, "#")
		return fmt.Sprintf("%s#legacyPackages.%s.%s", url, s, fragment), nil
	}

	return "", nil
}

// PackageAttributePath returns the short attribute path for a package which
// does not include packages/legacyPackages or the system name.
func (i *Input) PackageAttributePath() (string, error) {
	if i.IsDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		_, fragment, _ := strings.Cut(entry.Resolved, "#")
		return fragment, nil
	}
	return i.Fragment, nil
}

// FullPackageAttributePath returns the attribute path for a package. It is not
// always normalized which means it should not be used to compare packages.
// During happy paths (devbox packages and nix flakes that contains a fragment)
// it is much faster than NormalizedPackageAttributePath
func (i *Input) FullPackageAttributePath() (string, error) {
	if i.IsDevboxPackage() {
		reference, err := i.normalizedDevboxPackageReference()
		if err != nil {
			return "", err
		}
		_, fragment, _ := strings.Cut(reference, "#")
		return fragment, nil
	}
	return i.NormalizedPackageAttributePath()
}

// NormalizedPackageAttributePath returns an attribute path normalized by nix
// search. This is useful for comparing different attribute paths that may
// point to the same package. Note, it's an expensive call.
func (i *Input) NormalizedPackageAttributePath() (string, error) {
	var query string
	if i.isVersioned() {
		entry, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		query = entry.Resolved
	} else if i.IsDevboxPackage() {
		query = i.lockfile.LegacyNixpkgsPath(i.String())
	} else {
		query = i.String()
	}

	// We prefer search over just trying to parse the URL because search will
	// guarantee that the package exists for the current system.
	infos := search(query)

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

	if pkgExistsForAnySystem(query) {
		return "", usererr.New(
			"Package \"%s\" was found, but we're unable to build it for your system."+
				" You may need to choose another version or write a custom flake.",
			i.String(),
		)
	}

	return "", usererr.New("Package \"%s\" was not found", i.String())
}

func (i *Input) urlWithoutFragment() string {
	u := i.URL // get copy
	u.Fragment = ""
	return u.String()
}

func (i *Input) Hash() string {
	// For local flakes, use content hash of the flake.nix file to ensure
	// user always gets newest input.
	if i.IsLocal() {
		fileHash, _ := cuecfg.FileHash(filepath.Join(i.Path, "flake.nix"))
		if fileHash != "" {
			return fileHash[:6]
		}
	}
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) ValidateExists() (bool, error) {
	if i.isVersioned() && i.version() == "" {
		return false, usererr.New("No version specified for %q.", i.Path)
	}
	info, err := i.NormalizedPackageAttributePath()
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

	name, err := i.NormalizedPackageAttributePath()
	if err != nil {
		return false
	}
	otherName, err := other.NormalizedPackageAttributePath()
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

func (i *Input) Versioned() string {
	if featureflag.AutoLatest.Enabled() && i.IsDevboxPackage() && !i.isVersioned() {
		return i.Raw + "@latest"
	}
	return i.Raw
}

func (i *Input) EnsureNixpkgsPrefetched(w io.Writer) error {
	hash := i.hashFromNixPkgsURL()
	if hash == "" {
		return nil
	}
	return ensureNixpkgsPrefetched(w, hash)
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

func (i *Input) hashFromNixPkgsURL() string {
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

var nixPkgsRegex = regexp.MustCompile(`github:NixOS/nixpkgs/([^#]+).*`)

func HashFromNixPkgsURL(url string) string {
	matches := nixPkgsRegex.FindStringSubmatch(url)
	if len(matches) == 2 {
		return matches[1]
	}
	return ""
}
