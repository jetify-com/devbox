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

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/lock"
)

// Input represents a "package" added to the devbox.json config.
// The word "input" is used because it will be referenced in a generated flake.nix.
// A unique feature of flakes is that they have well-defined "inputs" and "outputs".
// This Input will be aggregated into a specific "flake input" (see shellgen.flakeInput).
type Input struct {
	url.URL
	lockfile lock.Locker

	// Raw is the devbox package name from the devbox.json config.
	// Raw has a few forms:
	// 1. Devbox Packages
	//    a. versioned packages
	//       examples:  go@1.20, python@latest
	//    b. any others?
	// 2. Local
	//    flakes in a relative sub-directory
	//    example: ./local_flake_subdir#myPackage
	// 3. Github
	//    remote flakes with raw name starting with `Github:`
	//    example: github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello
	Raw string

	normalizedPackageAttributePathCache string // memoized value from normalizedPackageAttributePath()
}

// InputsFromStrings constructs Input from the list of package names provided.
// These names correspond to devbox packages from the devbox.json config.
func InputsFromStrings(rawNames []string, l lock.Locker) []*Input {
	inputs := []*Input{}
	for _, rawName := range rawNames {
		inputs = append(inputs, InputFromString(rawName, l))
	}
	return inputs
}

// InputsFromStrings constructs Input from the raw name provided.
// The raw name corresponds to a devbox package from the devbox.json config.
func InputFromString(raw string, locker lock.Locker) *Input {
	// We ignore the error because... TODO @mikeland why?
	inputURL, _ := url.Parse(raw)

	// This handles local flakes in a relative path.
	// `raw` will be of the form `path:./local_flake_subdir#myPackage`
	// for which path:<empty>, opaque:./local_subdir, and scheme:path
	if inputURL.Path == "" && inputURL.Opaque != "" && inputURL.Scheme == "path" {
		// This normalizes url paths to be absolute. It also ensures all
		// path urls have a single slash (instead of possibly 3 slashes)
		normalizedURL := "path:" + filepath.Join(locker.ProjectDir(), inputURL.Opaque)
		if inputURL.Fragment != "" {
			normalizedURL += "#" + inputURL.Fragment
		}
		inputURL, _ = url.Parse(normalizedURL)
	}
	return &Input{*inputURL, locker, raw, ""}
}

// InputFromProfileItem sets the raw Input as the `item`'s unlockedReference i.e.
// the flake reference and output attribute path used at install time.
func InputFromProfileItem(item *NixProfileListItem, locker lock.Locker) *Input {
	return InputFromString(item.unlockedReference, locker)
}

// isLocal specifies whether this input is a local flake.
// Usually, this is of the form: `path:./local_flake_subdir#myPackage`
func (i *Input) isLocal() bool {
	// Technically flakes allows omitting the scheme for local absolute paths, but
	// we don't support that (yet).
	return i.Scheme == "path"
}

// isDevboxPackage specifies whether this input is a `canonicalName@version` nix
// package defined in a devbox.json config. This is in contrast to a "nix" package
// that can also be a flake or a legacy attribute path.
func (i *Input) isDevboxPackage() bool {
	return i.Scheme == ""
}

// isGithub specifies whether this input is a remote flake hosted on a github repository.
// example: github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello
func (i *Input) isGithub() bool {
	return i.Scheme == "github"
}

var inputNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]+")

// FlakeInputName refers to the name of this Input that will be used in the
// generated flake.nix. It is unique, and a slug is appended to avoid collisions.
//
// Note, that input name has nothing to do with the package name or flake name
// that it may be referencing. That is the Input.raw field.
func (i *Input) FlakeInputName() string {
	result := ""
	if i.isLocal() {
		result = filepath.Base(i.Path) + "-" + i.Hash()
	} else if i.isGithub() {
		result = "gh-" + strings.Join(strings.Split(i.Opaque, "/"), "-")
	} else if url := i.URLForFlakeInput(); IsGithubNixpkgsURL(url) {
		commitHash := HashFromNixPkgsURL(url)
		if len(commitHash) > 6 {
			commitHash = commitHash[0:6]
		}
		result = "nixpkgs-" + commitHash
	} else {
		result = i.String() + "-" + i.Hash()
	}

	// replace all non-alphanumeric with dashes
	return inputNameRegex.ReplaceAllString(result, "-")
}

// URLForFlakeInput is the url to be used as the input in the generated flake.nix
func (i *Input) URLForFlakeInput() string {
	if i.isDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.Raw)
		if err != nil {
			panic(err)
			// TODO(landau): handle error
		}
		withoutFragment, _, _ := strings.Cut(entry.Resolved, "#")
		return withoutFragment
	}
	return i.urlWithoutFragment()
}

// URLForInstall is used during `nix profile install`.
// The key difference with URLForFlakeInput is that it has a suffix of `#attributePath`
func (i *Input) URLForInstall() (string, error) {
	if i.isDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.Raw)
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
	if !i.isDevboxPackage() {
		return "", nil
	}

	path := ""
	if i.isVersioned() {
		entry, err := i.lockfile.Resolve(i.Raw)
		if err != nil {
			return "", err
		}
		path = entry.Resolved
	} else if i.isDevboxPackage() {
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
	if i.isDevboxPackage() {
		entry, err := i.lockfile.Resolve(i.Raw)
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
	if i.isDevboxPackage() {
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
// point to the same package. Note, it may be an expensive call.
func (i *Input) NormalizedPackageAttributePath() (string, error) {
	if i.normalizedPackageAttributePathCache != "" {
		return i.normalizedPackageAttributePathCache, nil
	}
	path, err := i.normalizePackageAttributePath()
	if err != nil {
		return path, err
	}
	i.normalizedPackageAttributePathCache = path
	return i.normalizedPackageAttributePathCache, nil
}

// normalizePackageAttributePath calls nix search to find the normalized attribute
// path. It is an expensive call (~100ms).
func (i *Input) normalizePackageAttributePath() (string, error) {
	var query string
	if i.isDevboxPackage() {
		if i.isVersioned() {
			entry, err := i.lockfile.Resolve(i.Raw)
			if err != nil {
				return "", err
			}
			query = entry.Resolved
		} else {
			query = i.lockfile.LegacyNixpkgsPath(i.String())
		}
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
	if i.isLocal() {
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

func (i *Input) Equals(other *Input) bool {
	if i.String() == other.String() {
		return true
	}

	// check inputs without fragments as optimization. Next step is expensive
	if i.URLForFlakeInput() != other.URLForFlakeInput() {
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
	if !i.isDevboxPackage() {
		return ""
	}
	name, _, _ := strings.Cut(i.Path, "@")
	return name
}

func (i *Input) Versioned() string {
	if i.isDevboxPackage() && !i.isVersioned() {
		return i.Raw + "@latest"
	}
	return i.Raw
}

func (i *Input) IsLegacy() bool {
	return i.isDevboxPackage() && !i.isVersioned()
}

func (i *Input) LegacyToVersioned() string {
	if !i.IsLegacy() {
		return i.Raw
	}
	return i.Raw + "@latest"
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
	if !i.isDevboxPackage() {
		return ""
	}
	_, version, _ := strings.Cut(i.Path, "@")
	return version
}

func (i *Input) isVersioned() bool {
	return i.isDevboxPackage() && strings.Contains(i.Path, "@")
}

func (i *Input) hashFromNixPkgsURL() string {
	return HashFromNixPkgsURL(i.URLForFlakeInput())
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
