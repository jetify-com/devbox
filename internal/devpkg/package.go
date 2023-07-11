// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/ux"
)

// Package represents a "package" added to the devbox.json config.
// A unique feature of flakes is that they have well-defined "inputs" and "outputs".
// This Package will be aggregated into a specific "flake input" (see shellgen.flakeInput).
type Package struct {
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

// PackageFromStrings constructs Package from the list of package names provided.
// These names correspond to devbox packages from the devbox.json config.
func PackageFromStrings(rawNames []string, l lock.Locker) []*Package {
	packages := []*Package{}
	for _, rawName := range rawNames {
		packages = append(packages, PackageFromString(rawName, l))
	}
	return packages
}

// PackageFromString constructs Package from the raw name provided.
// The raw name corresponds to a devbox package from the devbox.json config.
func PackageFromString(raw string, locker lock.Locker) *Package {
	// TODO: We should handle this error
	// TODO: URL might not be best representation since most packages are not urls
	pkgURL, _ := url.Parse(raw)

	// This handles local flakes in a relative path.
	// `raw` will be of the form `path:./local_flake_subdir#myPackage`
	// for which path:<empty>, opaque:./local_subdir, and scheme:path
	if pkgURL.Path == "" && pkgURL.Opaque != "" && pkgURL.Scheme == "path" {
		// This normalizes url paths to be absolute. It also ensures all
		// path urls have a single slash (instead of possibly 3 slashes)
		normalizedURL := "path:" + filepath.Join(locker.ProjectDir(), pkgURL.Opaque)
		if pkgURL.Fragment != "" {
			normalizedURL += "#" + pkgURL.Fragment
		}
		pkgURL, _ = url.Parse(normalizedURL)
	}

	return &Package{*pkgURL, locker, raw, ""}
}

// isLocal specifies whether this package is a local flake.
// Usually, this is of the form: `path:./local_flake_subdir#myPackage`
func (p *Package) isLocal() bool {
	// Technically flakes allows omitting the scheme for local absolute paths, but
	// we don't support that (yet).
	return p.Scheme == "path"
}

// isDevboxPackage specifies whether this package is a devbox package. Devbox
// packages have the format `canonicalName@version`and can be resolved by devbox
// search. This also returns true for legacy packages which are just an
// attribute path. An explicit flake reference is _not_ a devbox package.
func (p *Package) isDevboxPackage() bool {
	return p.Scheme == ""
}

// isGithub specifies whether this Package is referenced by a remote flake
// hosted on a github repository.
// example: github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello
func (p *Package) isGithub() bool {
	return p.Scheme == "github"
}

var inputNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]+")

// FlakeInputName generates a name for the input that will be used in the
// generated flake.nix to import this package. This name must be unique in that
// flake so we attach a hash to (quasi) ensure uniqueness.
// Input name will be different from raw package name
func (p *Package) FlakeInputName() string {
	result := ""
	if p.isLocal() {
		result = filepath.Base(p.Path) + "-" + p.Hash()
	} else if p.isGithub() {
		result = "gh-" + strings.Join(strings.Split(p.Opaque, "/"), "-")
	} else if url := p.URLForFlakeInput(); nix.IsGithubNixpkgsURL(url) {
		commitHash := nix.HashFromNixPkgsURL(url)
		if len(commitHash) > 6 {
			commitHash = commitHash[0:6]
		}
		result = "nixpkgs-" + commitHash
	} else {
		result = p.String() + "-" + p.Hash()
	}

	// replace all non-alphanumeric with dashes
	return inputNameRegex.ReplaceAllString(result, "-")
}

// URLForFlakeInput returns the input url to be used in a flake.nix file. This
// input can be used to import the package.
func (p *Package) URLForFlakeInput() string {
	if p.isDevboxPackage() {
		entry, err := p.lockfile.Resolve(p.Raw)
		if err != nil {
			panic(err)
			// TODO(landau): handle error
		}
		withoutFragment, _, _ := strings.Cut(entry.Resolved, "#")
		return withoutFragment
	}
	return p.urlWithoutFragment()
}

// Installable for this package. Installable is a nix concept defined here:
// https://nixos.org/manual/nix/stable/command-ref/new-cli/nix.html#installables
func (p *Package) Installable() (string, error) {
	inCache, err := p.IsInBinaryCache()
	if err != nil {
		return "", err
	}

	if inCache {
		// TODO savil: change to ContentAddressablePath?
		installable, err := p.InputAddressedPath()
		if err != nil {
			return "", err
		}
		return installable, nil
	}

	installable, err := p.urlForInstall()
	if err != nil {
		return "", err
	}
	return installable, nil
}

// urlForInstall is used during `nix profile install`.
// The key difference with URLForFlakeInput is that it has a suffix of
// `#attributePath`
func (p *Package) urlForInstall() (string, error) {
	if p.isDevboxPackage() {
		entry, err := p.lockfile.Resolve(p.Raw)
		if err != nil {
			return "", err
		}
		return entry.Resolved, nil
	}
	attrPath, err := p.FullPackageAttributePath()
	if err != nil {
		return "", err
	}
	return p.urlWithoutFragment() + "#" + attrPath, nil
}

func (p *Package) NormalizedDevboxPackageReference() (string, error) {
	if !p.isDevboxPackage() {
		return "", nil
	}

	path := ""
	if p.isVersioned() {
		entry, err := p.lockfile.Resolve(p.Raw)
		if err != nil {
			return "", err
		}
		path = entry.Resolved
	} else if p.isDevboxPackage() {
		path = p.lockfile.LegacyNixpkgsPath(p.String())
	}

	if path != "" {
		s, err := nix.System()
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
func (p *Package) PackageAttributePath() (string, error) {
	if p.isDevboxPackage() {
		entry, err := p.lockfile.Resolve(p.Raw)
		if err != nil {
			return "", err
		}
		_, fragment, _ := strings.Cut(entry.Resolved, "#")
		return fragment, nil
	}
	return p.Fragment, nil
}

// FullPackageAttributePath returns the attribute path for a package. It is not
// always normalized which means it should not be used to compare packages.
// During happy paths (devbox packages and nix flakes that contains a fragment)
// it is much faster than NormalizedPackageAttributePath
func (p *Package) FullPackageAttributePath() (string, error) {
	if p.isDevboxPackage() {
		reference, err := p.NormalizedDevboxPackageReference()
		if err != nil {
			return "", err
		}
		_, fragment, _ := strings.Cut(reference, "#")
		return fragment, nil
	}
	return p.NormalizedPackageAttributePath()
}

// NormalizedPackageAttributePath returns an attribute path normalized by nix
// search. This is useful for comparing different attribute paths that may
// point to the same package. Note, it may be an expensive call.
func (p *Package) NormalizedPackageAttributePath() (string, error) {
	if p.normalizedPackageAttributePathCache != "" {
		return p.normalizedPackageAttributePathCache, nil
	}
	path, err := p.normalizePackageAttributePath()
	if err != nil {
		return path, err
	}
	p.normalizedPackageAttributePathCache = path
	return p.normalizedPackageAttributePathCache, nil
}

// normalizePackageAttributePath calls nix search to find the normalized attribute
// path. It is an expensive call (~100ms).
func (p *Package) normalizePackageAttributePath() (string, error) {
	var query string
	if p.isDevboxPackage() {
		if p.isVersioned() {
			entry, err := p.lockfile.Resolve(p.Raw)
			if err != nil {
				return "", err
			}
			query = entry.Resolved
		} else {
			query = p.lockfile.LegacyNixpkgsPath(p.String())
		}
	} else {
		query = p.String()
	}

	// We prefer search over just trying to parse the URL because search will
	// guarantee that the package exists for the current system.
	infos := nix.Search(query)

	if len(infos) == 1 {
		return lo.Keys(infos)[0], nil
	}

	// If ambiguous, try to find a default output
	if len(infos) > 1 && p.Fragment == "" {
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
			p.String(),
			outputs,
		)
	}

	if nix.PkgExistsForAnySystem(query) {
		return "", usererr.New(
			"Package \"%s\" was found, but we're unable to build it for your system."+
				" You may need to choose another version or write a custom flake.",
			p.String(),
		)
	}

	return "", usererr.New("Package \"%s\" was not found", p.String())
}

func (p *Package) urlWithoutFragment() string {
	u := p.URL // get copy
	u.Fragment = ""
	return u.String()
}

func (p *Package) Hash() string {
	// For local flakes, use content hash of the flake.nix file to ensure
	// user always gets newest flake.
	if p.isLocal() {
		fileHash, _ := cuecfg.FileHash(filepath.Join(p.Path, "flake.nix"))
		if fileHash != "" {
			return fileHash[:6]
		}
	}
	hasher := md5.New()
	hasher.Write([]byte(p.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (p *Package) ValidateExists() (bool, error) {
	if p.isVersioned() && p.version() == "" {
		return false, usererr.New("No version specified for %q.", p.Path)
	}

	inCache, err := p.IsInBinaryCache()
	if err != nil {
		return false, err
	}
	if inCache {
		return true, nil
	}

	info, err := p.NormalizedPackageAttributePath()
	return info != "", err
}

func (p *Package) Equals(other *Package) bool {
	if p.String() == other.String() {
		return true
	}

	// check inputs without fragments as optimization. Next step is expensive
	if p.URLForFlakeInput() != other.URLForFlakeInput() {
		return false
	}

	name, err := p.NormalizedPackageAttributePath()
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
func (p *Package) CanonicalName() string {
	if !p.isDevboxPackage() {
		return ""
	}
	name, _, _ := strings.Cut(p.Path, "@")
	return name
}

func (p *Package) Versioned() string {
	if p.isDevboxPackage() && !p.isVersioned() {
		return p.Raw + "@latest"
	}
	return p.Raw
}

func (p *Package) IsLegacy() bool {
	return p.isDevboxPackage() && !p.isVersioned() && p.lockfile.Source(p.Raw) == ""
}

func (p *Package) LegacyToVersioned() string {
	if !p.IsLegacy() {
		return p.Raw
	}
	return p.Raw + "@latest"
}

func (p *Package) EnsureNixpkgsPrefetched(w io.Writer) error {

	inCache, err := p.IsInBinaryCache()
	if err != nil {
		return err
	}
	if inCache {
		// We can skip prefetching nixpkgs, if this package is in the binary
		// cache store.
		return nil
	}

	hash := p.HashFromNixPkgsURL()
	if hash == "" {
		return nil
	}
	return nix.EnsureNixpkgsPrefetched(w, hash)
}

// version returns the version of the package
// it only applies to devbox packages
func (p *Package) version() string {
	if !p.isDevboxPackage() {
		return ""
	}
	_, version, _ := strings.Cut(p.Path, "@")
	return version
}

func (p *Package) isVersioned() bool {
	return p.isDevboxPackage() && strings.Contains(p.Path, "@")
}

func (p *Package) HashFromNixPkgsURL() string {
	return nix.HashFromNixPkgsURL(p.URLForFlakeInput())
}

// BinaryCache is the store from which to fetch this package's binaries.
// It is used as FromStore in builtins.fetchClosure.
const BinaryCache = "https://cache.nixos.org"

func (p *Package) IsInBinaryCache() (bool, error) {
	if !featureflag.RemoveNixpkgs.Enabled() {
		return false, nil
	}

	if !p.isVersioned() {
		return false, nil
	}

	entry, err := p.lockfile.Resolve(p.Raw)
	if err != nil {
		return false, err
	}

	userSystem, err := nix.System()
	if err != nil {
		return false, err
	}

	if entry.Systems == nil {
		return false, nil
	}

	// Check if the user's system's info is present in the lockfile
	_, ok := entry.Systems[userSystem]
	return ok, nil
}

// InputAddressedPath is the input-addressed path in /nix/store
// It is also the key in the BinaryCache for this package
func (p *Package) InputAddressedPath() (string, error) {
	if inCache, err := p.IsInBinaryCache(); err != nil {
		return "", err
	} else if !inCache {
		return "",
			errors.Errorf("Package %q cannot be fetched from binary cache store", p.Raw)
	}

	entry, err := p.lockfile.Resolve(p.Raw)
	if err != nil {
		return "", err
	}

	userSystem, err := nix.System()
	if err != nil {
		return "", err
	}

	sysInfo := entry.Systems[userSystem]
	return sysInfo.StorePath, nil
}

// ContentAddressedPath is the content-addressed form of Package.InputAddressedPath
func (p *Package) ContentAddressedPath() (string, error) {

	if inCache, err := p.IsInBinaryCache(); err != nil {
		return "", err
	} else if !inCache {
		return "",
			errors.Errorf("Package %q cannot be fetched from binary cache store", p.Raw)
	}

	entry, err := p.lockfile.Resolve(p.Raw)
	if err != nil {
		return "", err
	}

	userSystem, err := nix.System()
	if err != nil {
		return "", err
	}

	sysInfo := entry.Systems[userSystem]
	if sysInfo.CAStorePath != "" {
		return sysInfo.CAStorePath, nil
	}

	ux.Fwarning(
		os.Stderr,
		"calculating ca_store_path. This may be slow. "+
			"Run `devbox update` to speed this up for next time.\n",
	)
	localPath, err := nix.ContentAddressedStorePath(sysInfo.StorePath)
	if err != nil {
		return "", err
	}
	return localPath, err
}
