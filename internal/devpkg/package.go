// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/devbox/plugins"
)

// Package represents a "package" added to the devbox.json config.
// A unique feature of flakes is that they have well-defined "inputs" and "outputs".
// This Package will be aggregated into a specific "flake input" (see shellgen.flakeInput).
type Package struct {
	plugins.BuiltIn
	lockfile        lock.Locker
	IsDevboxPackage bool

	// If package triggers a built-in plugin, setting this to true will disable it.
	// If package does not trigger plugin, this will have no effect.
	DisablePlugin bool

	// installable is the flake attribute that the package resolves to.
	// When it gets set depends on the original package string:
	//
	// - If the parsed package string is unambiguously a flake installable
	//   (not "name" or "name@version"), then it is set immediately.
	// - Otherwise, it's set after calling resolve.
	//
	// This is done for performance reasons. Some commands don't require the
	// fully-resolved package, so we don't want to waste time computing it.
	installable flake.Installable

	// resolve resolves a Devbox package string to a Nix installable.
	//
	// - If the package exists in the lockfile, it resolves to the
	//   lockfile's installable.
	// - If the package doesn't exist in the lockfile, it resolves to the
	//   installable returned by the search index (/v1/resolve).
	//
	// After resolving the installable, it also sets storePath when the
	// package exists in the Nix binary cache.
	//
	// For flake packages (non-devbox packages), resolve is a no-op.
	resolve func() error

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

	// Outputs is a list of outputs to build from the package's derivation.
	// If empty, the default output is used.
	//
	// Note the distinction: these are user-selected outputs, whereas the lockfile has outputs
	// whose store-paths are available outputs with store-paths.
	Outputs []string

	// PatchGlibc applies a function to the package's derivation that
	// patches any ELF binaries to use the latest version of nixpkgs#glibc.
	PatchGlibc bool

	// AllowInsecure are a list of nix packages that are whitelisted to be
	// installed even if they are marked as insecure.
	AllowInsecure []string

	// isInstallable is true if the package may be enabled on the current platform.
	isInstallable bool

	normalizedPackageAttributePathCache string // memoized value from normalizedPackageAttributePath()
}

// PackagesFromStringsWithDefaults constructs Package from the list of package names provided.
// These names correspond to devbox packages from the devbox.json config.
func PackagesFromStringsWithDefaults(rawNames []string, l lock.Locker) []*Package {
	packages := []*Package{}
	for _, rawName := range rawNames {
		pkg := PackageFromStringWithDefaults(rawName, l)
		packages = append(packages, pkg)
	}
	return packages
}

func PackagesFromStringsWithOptions(rawNames []string, l lock.Locker, opts devopt.AddOpts) []*Package {
	packages := []*Package{}
	for _, name := range rawNames {
		packages = append(packages, PackageFromStringWithOptions(name, l, opts))
	}
	return packages
}

func PackagesFromConfig(config *devconfig.Config, l lock.Locker) []*Package {
	result := []*Package{}
	for _, cfgPkg := range config.Packages() {
		pkg := newPackage(cfgPkg.VersionedName(), cfgPkg.IsEnabledOnPlatform(), l)
		pkg.DisablePlugin = cfgPkg.DisablePlugin
		pkg.PatchGlibc = cfgPkg.PatchGlibc && nix.SystemIsLinux()
		pkg.Outputs = initOutputsField(cfgPkg.Outputs)
		pkg.AllowInsecure = cfgPkg.AllowInsecure
		result = append(result, pkg)
	}
	return result
}

func PackageFromStringWithDefaults(raw string, locker lock.Locker) *Package {
	return newPackage(raw, true /*isInstallable*/, locker)
}

func PackageFromStringWithOptions(raw string, locker lock.Locker, opts devopt.AddOpts) *Package {
	pkg := PackageFromStringWithDefaults(raw, locker)
	pkg.DisablePlugin = opts.DisablePlugin
	pkg.PatchGlibc = opts.PatchGlibc
	pkg.Outputs = initOutputsField(opts.Outputs)
	pkg.AllowInsecure = opts.AllowInsecure
	return pkg
}

func newPackage(raw string, isInstallable bool, locker lock.Locker) *Package {
	pkg := &Package{
		Raw:           raw,
		lockfile:      locker,
		isInstallable: isInstallable,
		Outputs:       initOutputsField([]string{}),
	}

	// The raw string is either a Devbox package ("name" or "name@version")
	// or it's a flake installable. In some cases they're ambiguous
	// ("nixpkgs" is a devbox package and a flake). When that happens, we
	// assume a Devbox package.
	parsed, err := flake.ParseInstallable(raw)
	if err != nil || isAmbiguous(raw, parsed) {
		pkg.IsDevboxPackage = true
		pkg.resolve = sync.OnceValue(func() error { return resolve(pkg) })
		return pkg
	}

	// We currently don't lock flake references in devbox.lock, so there's
	// nothing to resolve.
	pkg.resolve = sync.OnceValue(func() error { return nil })
	pkg.setInstallable(parsed, locker.ProjectDir())
	return pkg
}

// UseDefaultOutput is a special signifier to use the default outputs of a package.
// It is used to indicate that the user hasn't explicitly specified the outputs they want.
const UseDefaultOutput = "__useDefaultOutput__"

// initOutputsField initializes the outputs field of a package. It is meant to be used in the
// Package struct constructor functions
func initOutputsField(selectedOutputs []string) []string {
	outputs := []string{UseDefaultOutput}
	if len(selectedOutputs) > 0 {
		outputs = selectedOutputs
	}
	return outputs
}

// isAmbiguous returns true if a package string could be a Devbox package or
// a flake installable. For example, "nixpkgs" is both a Devbox package and a
// flake.
func isAmbiguous(raw string, parsed flake.Installable) bool {
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

// resolve is the implementation of Package.resolve, where it is wrapped in a
// sync.OnceValue function. It should not be called directly.
func resolve(pkg *Package) error {
	resolved, err := pkg.lockfile.Resolve(pkg.Raw)
	if err != nil {
		return err
	}
	parsed, err := flake.ParseInstallable(resolved.Resolved)
	if err != nil {
		return err
	}

	// TODO savil. Check with Greg about setting the user-specified outputs
	// somehow here.
	// NOTE: The below code fails with the php testscript.

	pkg.setInstallable(parsed, pkg.lockfile.ProjectDir())
	return nil
}

func (p *Package) setInstallable(i flake.Installable, projectDir string) {
	if i.Ref.Type == flake.TypePath && !filepath.IsAbs(i.Ref.Path) {
		i.Ref.Path = filepath.Join(projectDir, i.Ref.Path)
	}
	p.installable = i
}

var inputNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]+")

// FlakeInputName generates a name for the input that will be used in the
// generated flake.nix to import this package. This name must be unique in that
// flake so we attach a hash to (quasi) ensure uniqueness.
// Input name will be different from raw package name
func (p *Package) FlakeInputName() string {
	_ = p.resolve()

	result := ""
	switch p.installable.Ref.Type {
	case flake.TypePath:
		result = filepath.Base(p.installable.Ref.Path) + "-" + p.Hash()
	case flake.TypeGitHub:
		isNixOS := strings.ToLower(p.installable.Ref.Owner) == "nixos"
		isNixpkgs := isNixOS && strings.ToLower(p.installable.Ref.Repo) == "nixpkgs"
		if isNixpkgs && p.IsDevboxPackage {
			commitHash := nix.HashFromNixPkgsURL(p.installable.Ref.String())
			result = "nixpkgs-" + commitHash[:min(6, len(commitHash))]
		} else {
			result = "gh-" + p.installable.Ref.Owner + "-" + p.installable.Ref.Repo
			if p.installable.Ref.Rev != "" {
				result += "-" + p.installable.Ref.Rev
			} else if p.installable.Ref.Ref != "" {
				result += "-" + p.installable.Ref.Ref
			}
		}
	default:
		result = p.installable.String() + "-" + p.Hash()
	}

	// replace all non-alphanumeric with dashes
	return inputNameRegex.ReplaceAllString(result, "-")
}

// URLForFlakeInput returns the input url to be used in a flake.nix file. This
// input can be used to import the package.
func (p *Package) URLForFlakeInput() string {
	if err := p.resolve(); err != nil {
		// TODO(landau): handle error
		panic(err)
	}
	return p.installable.Ref.String()
}

// IsInstallable returns whether this package is installable. Not to be confused
// with the Installable() method which returns the corresponding nix concept.
func (p *Package) IsInstallable() bool {
	return p.isInstallable
}

// Installable for this package. Installable is a nix concept defined here:
// https://nixos.org/manual/nix/stable/command-ref/new-cli/nix.html#installables
func (p *Package) Installable() (string, error) {
	outputs, err := p.GetOutputNames()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "Package.Installable: outputs for package %s: %v\n", p.Raw, outputs)
	installables := []string{}
	for _, output := range outputs {
		i, err := p.InstallableForOutput(output)
		if err != nil {
			return "", err
		}
		installables = append(installables, i)
	}
	if len(installables) == 0 {
		// This means that the package is not in the binary cache
		// OR it is a flake (??)
		installable, err := p.urlForInstall()
		if err != nil {
			return "", err
		}
		return installable, nil
	}
	// TODO savil: return all installables
	return installables[0], nil
}

func (p *Package) InstallableForOutput(output string) (string, error) {
	inCache, err := p.IsOutputInBinaryCache(output)
	if err != nil {
		return "", err
	}

	if inCache {
		fmt.Fprintf(os.Stderr, "InstallableForOutput: Package %s output %s is in the binary cache\n", p.Raw, output)
		installable, err := p.InputAddressedPathForOutput(output)
		if err != nil {
			return "", err
		}
		return installable, nil
	}

	// TODO savil: make work for output
	installable, err := p.urlForInstall()
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "InstallableForOutput: Package %s output %s is NOT in the binary cache\n", p.Raw, output)
	fmt.Fprintf(os.Stderr, "InstallableForOutput: Package %s output %s installable: %s \n", p.Raw, output, installable)
	return installable, nil
}

// FlakeInstallable returns a flake installable. The raw string must contain
// a valid flake reference parsable by ParseFlakeRef, optionally followed by an
// #attrpath and/or an ^output.
func (p *Package) FlakeInstallable() (flake.Installable, error) {
	return flake.ParseInstallable(p.Raw)
}

// urlForInstall is used during `nix profile install`.
// The key difference with URLForFlakeInput is that it has a suffix of
// `#attributePath`
func (p *Package) urlForInstall() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	return p.installable.String(), nil
}

func (p *Package) NormalizedDevboxPackageReference() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	if p.installable.AttrPath == "" {
		return "", nil
	}
	clone := p.installable
	clone.AttrPath = fmt.Sprintf("legacyPackages.%s.%s", nix.System(), clone.AttrPath)
	return clone.String(), nil
}

// PackageAttributePath returns the short attribute path for a package which
// does not include packages/legacyPackages or the system name.
func (p *Package) PackageAttributePath() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}
	return p.installable.AttrPath, nil
}

// FullPackageAttributePath returns the attribute path for a package. It is not
// always normalized which means it should not be used to compare packages.
// During happy paths (devbox packages and nix flakes that contains a fragment)
// it is much faster than NormalizedPackageAttributePath
func (p *Package) FullPackageAttributePath() (string, error) {
	if p.IsDevboxPackage {
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
// path. It may be an expensive call (~100ms).
func (p *Package) normalizePackageAttributePath() (string, error) {
	if err := p.resolve(); err != nil {
		return "", err
	}

	query := p.installable.String()
	if query == "" {
		query = p.Raw
	}

	// We prefer nix.Search over just trying to parse the package's "URL" because
	// nix.Search will guarantee that the package exists for the current system.
	var infos map[string]*nix.Info
	var err error
	if p.IsDevboxPackage && !p.IsRunX() {
		// Perf optimization: For queries of the form nixpkgs/<commit>#foo, we can
		// use a nix.Search cache.
		//
		// This will be slow if its the first time on the user's machine that this
		// query is running. Otherwise, it will be cached and fast.
		if infos, err = nix.SearchNixpkgsAttribute(query); err != nil {
			return "", err
		}
	} else {
		// fallback to the slow but generalized nix.Search
		if infos, err = nix.Search(query); err != nil {
			return "", err
		}
	}

	if len(infos) == 1 {
		return lo.Keys(infos)[0], nil
	}

	// If ambiguous, try to find a default output
	if len(infos) > 1 && p.installable.AttrPath == "" {
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
			p.Raw,
			outputs,
		)
	}

	if nix.PkgExistsForAnySystem(query) {
		return "", usererr.WithUserMessage(
			ErrCannotBuildPackageOnSystem,
			"Package \"%s\" was found, but we're unable to build it for your system."+
				" You may need to choose another version or write a custom flake.",
			p.Raw,
		)
	}

	return "", usererr.New("Package \"%s\" was not found", p.Raw)
}

var ErrCannotBuildPackageOnSystem = errors.New("unable to build for system")

func (p *Package) Hash() string {
	sum := ""
	if p.installable.Ref.Type == flake.TypePath {
		// For local flakes, use content hash of the flake.nix file to ensure
		// user always gets newest flake.
		sum, _ = cachehash.File(filepath.Join(p.installable.Ref.Path, "flake.nix"))
	}

	if sum == "" {
		sum, _ = cachehash.Bytes([]byte(p.installable.String()))
	}
	return sum[:min(len(sum), 6)]
}

// Equals compares two Packages. This may be an expensive operation since it
// may have to normalize a Package's attribute path, which may require a network
// call.
func (p *Package) Equals(other *Package) bool {
	if p.Raw == other.Raw || p.installable == other.installable {
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
	if !p.IsDevboxPackage {
		return ""
	}
	name, _, _ := strings.Cut(p.Raw, "@")
	return name
}

func (p *Package) Versioned() string {
	if p.IsDevboxPackage && !p.isVersioned() {
		return p.Raw + "@latest"
	}
	return p.Raw
}

func (p *Package) IsLegacy() bool {
	return p.IsDevboxPackage && !p.isVersioned() && p.lockfile.Get(p.Raw).GetSource() == ""
}

func (p *Package) LegacyToVersioned() string {
	if !p.IsLegacy() {
		return p.Raw
	}
	return p.Raw + "@latest"
}

// EnsureNixpkgsPrefetched will prefetch flake for the nixpkgs registry for the package.
// This is an internal method, and should not be called directly.
func EnsureNixpkgsPrefetched(ctx context.Context, w io.Writer, pkgs []*Package) error {
	for _, input := range pkgs {
		if err := input.ensureNixpkgsPrefetched(w); err != nil {
			return err
		}
	}
	return nil
}

// ensureNixpkgsPrefetched should be called via the public EnsureNixpkgsPrefetched.
// See function comment there.
func (p *Package) ensureNixpkgsPrefetched(w io.Writer) error {
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
	if !p.IsDevboxPackage {
		return ""
	}
	_, version, _ := strings.Cut(p.Raw, "@")
	return version
}

func (p *Package) isVersioned() bool {
	return p.IsDevboxPackage && strings.Contains(p.Raw, "@")
}

func (p *Package) HashFromNixPkgsURL() string {
	return nix.HashFromNixPkgsURL(p.URLForFlakeInput())
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

	sysInfo := entry.Systems[nix.System()]
	outputs := sysInfo.DefaultOutputs()

	// TODO return an array of outputs
	return p.InputAddressedPathForOutput(outputs[0].Name)
}

func (p *Package) InputAddressedPathForOutput(output string) (string, error) {
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

	sysInfo := entry.Systems[nix.System()]
	for _, out := range sysInfo.Outputs {
		if out.Name == output {
			return out.Path, nil
		}
	}
	return "", errors.Errorf("Output %q not found for package %q", output, p.Raw)
}

func (p *Package) HasAllowInsecure() bool {
	return len(p.AllowInsecure) > 0
}

// StoreName returns the last section of the store path. Example:
// /nix/store/abc123-foo-1.0.0 -> foo-1.0.0
// Warning, this is probably slowish. If you need to call this multiple times,
// consider caching the result.
func (p *Package) StoreName() (string, error) {
	u, err := p.urlForInstall()
	if err != nil {
		return "", err
	}
	name, err := nix.EvalPackageName(u)
	if err != nil {
		return "", err
	}
	return name, nil
}

func (p *Package) EnsureUninstallableIsInLockfile() error {
	// TODO savil: Should !p.isInstallable() be the opposite i.e. p.IsInstallable()?
	// TODO savil: Do we need the IsDevboxPackage check here?
	if !p.IsInstallable() || !p.IsDevboxPackage {
		return nil
	}
	_, err := p.lockfile.Resolve(p.Raw)
	return err
}

func (p *Package) IsRunX() bool {
	return pkgtype.IsRunX(p.Raw)
}

func (p *Package) IsNix() bool {
	return IsNix(p, 0)
}

func (p *Package) RunXPath() string {
	return strings.TrimPrefix(p.Raw, pkgtype.RunXPrefix)
}

func (p *Package) String() string {
	if p.installable.AttrPath != "" {
		return p.installable.AttrPath
	}
	return p.Raw
}

func IsNix(p *Package, _ int) bool {
	return !p.IsRunX()
}

func IsRunX(p *Package, _ int) bool {
	return p.IsRunX()
}

func (p *Package) DocsURL() string {
	if p.IsRunX() {
		path, _, _ := strings.Cut(p.RunXPath(), "@")
		return fmt.Sprintf("https://www.github.com/%s", path)
	}
	if p.IsDevboxPackage {
		return fmt.Sprintf("https://www.nixhub.io/packages/%s", p.CanonicalName())
	}
	return ""
}

// GetOutputNames returns the names of the nix package outputs.
// It may be empty if the package is not in the lockfile.
func (p *Package) GetOutputNames() ([]string, error) {
	if p.IsRunX() {
		return []string{}, nil
	}

	// if p.Outputs has user specified outputs:
	if len(p.Outputs) > 1 || p.Outputs[0] != UseDefaultOutput {
		fmt.Fprintf(os.Stderr, "Returning user specified outputs: %v\n", p.Outputs)
		return p.Outputs, nil
	}
	// else, get the default outputs from the lockfile

	sysInfo, err := p.sysInfoIfExists()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error getting sysInfo: %v\n", err)
		return []string{}, err
	} else if sysInfo == nil {
		fmt.Fprintf(os.Stderr, "No sysInfo found for pkg %s\n", p.Raw)
		// TODO should this be "out" or empty?
		return []string{}, nil
	}

	names := []string{}
	for _, output := range sysInfo.DefaultOutputs() {
		names = append(names, output.Name)
	}
	fmt.Fprintf(os.Stderr, "for package %s returning default outputs : %v\n", p.Raw, names)
	return names, nil
}
