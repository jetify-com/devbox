// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"context"
	"io/fs"
	"maps"
	"path/filepath"
	"slices"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/cachehash"
	"go.jetify.com/devbox/internal/devpkg/pkgtype"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/searcher"
	"go.jetify.com/devbox/nix/flake"
	"go.jetify.com/pkg/runx/impl/types"
	"golang.org/x/sync/singleflight"

	"go.jetify.com/devbox/internal/cuecfg"
)

const lockFileVersion = "1"

// Lightly inspired by package-lock.json
//
// File's methods are safe to call concurrently. Code that reads or writes
// Packages directly must not run while other goroutines call File methods.
type File struct {
	devboxProject `json:"-"`

	// mu guards Packages and the entries in it. It is never held while
	// resolving a package, which can make network requests or run nix.
	mu sync.Mutex
	// resolving dedupes concurrent resolutions of the same package.
	resolving singleflight.Group

	LockFileVersion string `json:"lockfile_version"`

	// Packages is keyed by "canonicalName@version"
	Packages map[string]*Package `json:"packages"`
}

func GetFile(project devboxProject) (*File, error) {
	lockFile := &File{
		devboxProject: project,

		LockFileVersion: lockFileVersion,
		Packages:        map[string]*Package{},
	}
	err := cuecfg.ParseFile(lockFilePath(project.ProjectDir()), lockFile)
	if errors.Is(err, fs.ErrNotExist) {
		return lockFile, nil
	}
	if err != nil {
		return nil, err
	}

	// If the lockfile has legacy StorePath fields, we need to convert them to the new format
	ensurePackagesHaveOutputs(lockFile.Packages)

	return lockFile, nil
}

func (f *File) Add(pkgs ...string) error {
	for _, p := range pkgs {
		if _, err := f.Resolve(p); err != nil {
			return err
		}
	}
	return f.Save()
}

func (f *File) Remove(pkgs ...string) error {
	f.mu.Lock()
	for _, p := range pkgs {
		delete(f.Packages, p)
	}
	f.mu.Unlock()
	return f.Save()
}

// Resolve updates the in memory copy for performance but does not write to disk
// This avoids writing values that may need to be removed in case of error.
//
// Resolve is safe to call concurrently. The lockfile's lock is only held to read
// and write the entry; the search or flake request runs outside of it.
// Concurrent calls for the same package share a single request.
func (f *File) Resolve(pkg string) (*Package, error) {
	if entry := f.resolvedEntry(pkg); entry != nil {
		return entry, nil
	}
	v, err, _ := f.resolving.Do(pkg, func() (any, error) {
		// Another call may have finished resolving pkg between the check
		// above and joining this group.
		if entry := f.resolvedEntry(pkg); entry != nil {
			return entry, nil
		}
		locked, err := f.fetchLockEntry(pkg)
		if err != nil {
			return nil, err
		}
		f.mu.Lock()
		defer f.mu.Unlock()
		f.Packages[pkg] = locked
		return locked, nil
	})
	if err != nil {
		return nil, err
	}
	return v.(*Package), nil
}

// resolvedEntry returns the lockfile entry for pkg if it's already resolved, or
// nil otherwise.
func (f *File) resolvedEntry(pkg string) *Package {
	f.mu.Lock()
	defer f.mu.Unlock()
	if entry, ok := f.Packages[pkg]; ok && entry.Resolved != "" {
		return entry
	}
	return nil
}

// fetchLockEntry computes a new lockfile entry for pkg without storing it. It
// must not be called with f.mu held.
func (f *File) fetchLockEntry(pkg string) (*Package, error) {
	locked := &Package{}
	_, _, versioned := searcher.ParseVersionedPackage(pkg)
	if pkgtype.IsRunX(pkg) || versioned || pkgtype.IsFlake(pkg) {
		resolved, err := f.FetchResolvedPackage(pkg, false)
		if err != nil {
			return nil, err
		}
		if resolved != nil {
			locked = resolved
		}
	} else if IsLegacyPackage(pkg) {
		// These are legacy packages without a version. Resolve to nixpkgs with
		// whatever hash is in the devbox.json
		locked = &Package{
			Resolved: flake.Installable{
				Ref:      f.Stdenv(),
				AttrPath: pkg,
			}.String(),
			Source: nixpkgSource,
		}
	}
	return locked, nil
}

// TODO:
// Consider a design change to have the File struct match disk to make this system
// easier to reason about, and have isDirty() compare the in-memory struct to the
// on-disk struct.
//
// Proposal:
// 1. Have an OutputsRaw field and a method called Outputs() to access it.
// Outputs() will check if OutputsRaw is zero-value and fills it in from StorePath.
// 2. Then, in Save(), we can check if OutputsRaw is zero and fill it in prior to writing
// to disk.
func (f *File) Save() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	isDirty, err := f.isDirtyLocked()
	if err != nil {
		return err
	}
	if !isDirty {
		return nil
	}

	// In SystemInfo, preserve legacy StorePath field and clear out modern Outputs before writing
	// Reason: We want to update `devbox.lock` file only upon a user action
	// such as `devbox update` or `devbox add` or `devbox remove`.
	for pkgName, pkg := range f.Packages {
		for sys, sysInfo := range pkg.Systems {
			if sysInfo.outputIsFromStorePath {
				f.Packages[pkgName].Systems[sys].Outputs = nil
			}
		}
	}
	// We set back the Outputs, if needed, after writing the file, so that future
	// users of the `lock.File` struct will have the correct data.
	defer ensurePackagesHaveOutputs(f.Packages)

	return cuecfg.WriteFile(lockFilePath(f.devboxProject.ProjectDir()), f)
}

func (f *File) UpdateStdenv() error {
	if err := nix.ClearFlakeCache(f.devboxProject.Stdenv()); err != nil {
		return err
	}
	// Remove the old entry in memory only so that a failed resolve doesn't
	// leave the lockfile without a stdenv.
	stdenv := f.devboxProject.Stdenv().String()
	f.mu.Lock()
	old, hasOld := f.Packages[stdenv]
	delete(f.Packages, stdenv)
	f.mu.Unlock()
	if err := f.Add(stdenv); err != nil {
		if hasOld {
			f.mu.Lock()
			f.Packages[stdenv] = old
			f.mu.Unlock()
		}
		return err
	}
	return nil
}

// TODO: We should improve a few issues with this function:
// * It shared the same name as Devbox.Stdenv() which is confusing.
// * Since File implements DevboxProject, IDEs really struggle to accurately find call sites.
// (side note, we should remove DevboxProject interface)
// * This function forces a resolution of the stdenv flake which is slow and doesn't give us a
// chance to "prep" the user for some waiting.
// * Should we rename to Nixpkgs() ? Stdenv feels a bit ambiguous.
func (f *File) Stdenv() flake.Ref {
	unlocked := f.devboxProject.Stdenv()
	pkg, err := f.Resolve(unlocked.String())
	if err != nil {
		return unlocked
	}
	ref, err := flake.ParseRef(pkg.Resolved)
	if err != nil {
		return unlocked
	}
	return ref
}

func (f *File) Get(pkg string) *Package {
	return f.resolvedEntry(pkg)
}

func (f *File) HasAllowInsecurePackages() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, pkg := range f.Packages {
		if pkg.AllowInsecure {
			return true
		}
	}
	return false
}

// This probably belongs in input.go but can't add it there because it will
// create a circular dependency. We could move Input into own package.
func IsLegacyPackage(pkg string) bool {
	_, _, versioned := searcher.ParseVersionedPackage(pkg)
	return !versioned &&
		!strings.Contains(pkg, ":") &&
		// We don't support absolute paths without "path:" prefix, but adding here
		// just in case we ever do.
		// Landau note: I don't think we should support it, it's hard to read and a
		// bit ambiguous.
		!strings.HasPrefix(pkg, "/")
}

// Tidy ensures that the lockfile has the set of packages corresponding to the devbox.json config.
// It gets rid of older packages that are no longer needed.
func (f *File) Tidy() {
	keep := f.devboxProject.AllPackageNamesIncludingRemovedTriggerPackages()
	keep = append(keep, f.devboxProject.Stdenv().String())
	f.mu.Lock()
	defer f.mu.Unlock()
	maps.DeleteFunc(f.Packages, func(key string, pkg *Package) bool {
		return !slices.Contains(keep, key)
	})
}

// IsUpToDateAndInstalled returns true if the lockfile is up to date and the
// local hashes match, which generally indicates all packages are correctly
// installed and print-dev-env has been computed and cached.
func (f *File) IsUpToDateAndInstalled(isFish bool) (bool, error) {
	status, err := f.StateStatus(isFish)
	return status == StateUpToDate, err
}

// StateStatus is like IsUpToDateAndInstalled, but also reports whether only
// non-Nix inputs (env, init_hook, scripts, etc.) changed, in which case the
// Nix environment doesn't need to be recomputed.
func (f *File) StateStatus(isFish bool) (StateStatus, error) {
	if dirty, err := f.isDirty(); err != nil {
		return StateStale, err
	} else if dirty {
		return StateStale, nil
	}
	configHash, err := f.devboxProject.ConfigHash()
	if err != nil {
		return StateStale, err
	}
	nixConfigHash, err := f.devboxProject.NixConfigHash()
	if err != nil {
		return StateStale, err
	}
	return stateStatus(UpdateStateHashFileArgs{
		ProjectDir:    f.devboxProject.ProjectDir(),
		ConfigHash:    configHash,
		NixConfigHash: nixConfigHash,
		IsFish:        isFish,
	})
}

func (f *File) SetOutputsForPackage(pkg string, outputs []Output) error {
	p, err := f.Resolve(pkg)
	if err != nil {
		return err
	}
	f.mu.Lock()
	if p.Systems == nil {
		p.Systems = map[string]*SystemInfo{}
	}
	if p.Systems[nix.System()] == nil {
		p.Systems[nix.System()] = &SystemInfo{}
	}
	p.Systems[nix.System()].Outputs = outputs
	f.mu.Unlock()
	return f.Save()
}

func (f *File) isDirty() (bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.isDirtyLocked()
}

func (f *File) isDirtyLocked() (bool, error) {
	currentHash, err := cachehash.JSON(f)
	if err != nil {
		return false, err
	}
	fileSystemLockFile, err := GetFile(f.devboxProject)
	if err != nil {
		return false, err
	}
	filesystemHash, err := cachehash.JSON(fileSystemLockFile)
	if err != nil {
		return false, err
	}
	return currentHash != filesystemHash, nil
}

func lockFilePath(projectDir string) string {
	return filepath.Join(projectDir, "devbox.lock")
}

func ResolveRunXPackage(ctx context.Context, pkg string) (types.PkgRef, error) {
	ref, err := types.NewPkgRef(strings.TrimPrefix(pkg, pkgtype.RunXPrefix))
	if err != nil {
		return types.PkgRef{}, err
	}

	registry, err := pkgtype.RunXRegistry(ctx)
	if err != nil {
		return types.PkgRef{}, err
	}
	return registry.ResolveVersion(ref)
}
