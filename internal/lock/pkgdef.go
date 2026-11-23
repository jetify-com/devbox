// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"context"
	"errors"
	"maps"
	"time"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/pkgdef"
)

// resolvePkgDef resolves a "pkg:<ref>@<version>" package string to a
// lockfile entry: the concrete version plus the artifact URL and checksum
// for every system the definition supports.
//
// When refresh is false and the package already has an entry, a remote
// definition is pinned to the entry's definition_hash so that `devbox add`
// and `devbox install` never silently pick up a changed definition. Only
// `devbox update` (refresh=true) re-fetches it.
func (f *File) resolvePkgDef(ctx context.Context, raw string, refresh bool) (*Package, error) {
	ref, err := pkgdef.ParseRef(raw)
	if err != nil {
		return nil, usererr.WithUserMessage(err, "Invalid package %q", raw)
	}

	// Resolve can run concurrently for different packages; read the
	// existing entry under the lock (resolving runs outside of it).
	f.mu.Lock()
	existing := f.Packages[raw]
	f.mu.Unlock()
	return f.resolvePkgDefRef(ctx, ref, raw, existing, refresh)
}

// RelockPkgDef re-resolves the lockfile entry of a local package definition
// whose file changed since it was locked, keeping the locked version. A
// changed install recipe may point at another artifact, and the entry's
// URL and sha256 must follow it. The entry is updated in memory; callers
// save the lockfile.
func (f *File) RelockPkgDef(ctx context.Context, raw string) (*Package, error) {
	ref, err := pkgdef.ParseRef(raw)
	if err != nil {
		return nil, usererr.WithUserMessage(err, "Invalid package %q", raw)
	}
	f.mu.Lock()
	existing := f.Packages[raw]
	f.mu.Unlock()
	if existing != nil && existing.Version != "" {
		ref.Version = existing.Version
	}
	locked, err := f.resolvePkgDefRef(ctx, ref, raw, existing, false /*refresh*/)
	if err != nil {
		return nil, err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Packages[raw] = locked
	return locked, nil
}

func (f *File) resolvePkgDefRef(ctx context.Context, ref pkgdef.Ref, raw string, existing *Package, refresh bool) (*Package, error) {
	pinned := ""
	if !refresh && existing != nil {
		pinned = existing.DefinitionHash
	}

	client := pkgdef.NewClient()
	loaded, err := client.Load(ctx, ref, f.ProjectDir(), pinned)
	if err != nil {
		return nil, err
	}
	resolved, err := client.Resolve(ctx, loaded.Definition, ref.Version)
	if errors.Is(err, pkgdef.ErrVersionNotFound) {
		return nil, usererr.WithUserMessage(err, "Package %s not found", raw)
	}
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Resolved:       ref.WithVersion(resolved.Version).String(),
		Version:        resolved.Version,
		Source:         pkgdef.Source,
		DefinitionHash: loaded.Hash,
		Systems:        make(map[string]*SystemInfo, len(resolved.Artifacts)),
	}
	// Systems the vendor publishes no artifact for are absent from
	// Systems; installing on one of them is an error.
	for system, artifact := range resolved.Artifacts {
		info := &SystemInfo{URL: artifact.URL, Sha256: artifact.Sha256, Sha512: artifact.Sha512}
		// A definition without a checksum source gets its digest recorded
		// by the first install (trust on first use). Keep that digest when
		// re-resolving to the same artifact so `devbox update` doesn't
		// throw it away.
		if !artifact.HasDigest() && existing != nil && existing.Version == resolved.Version {
			if old := existing.Systems[system]; old != nil && old.URL == artifact.URL {
				info.Sha256 = old.Sha256
			}
		}
		pkg.Systems[system] = info
	}

	// LastModified is what multi-project lockfile sync uses to pick the
	// newest entry. Keep it when nothing changed so re-resolving does not
	// churn devbox.lock.
	pkg.LastModified = time.Now().UTC().Format(time.RFC3339)
	if existing != nil && existing.LastModified != "" &&
		existing.Version == pkg.Version &&
		existing.DefinitionHash == pkg.DefinitionHash &&
		maps.EqualFunc(existing.Systems, pkg.Systems, (*SystemInfo).Equals) {
		pkg.LastModified = existing.LastModified
	}
	return pkg, nil
}

// SetArtifactForPackage records the URL and sha256 of the artifact that was
// installed for system. It is used to fill in digests for definitions that
// have no checksum source.
func (f *File) SetArtifactForPackage(pkg, system, url, sha256 string) error {
	locked, err := f.Resolve(pkg)
	if err != nil {
		return err
	}
	if locked.Systems == nil {
		locked.Systems = map[string]*SystemInfo{}
	}
	if locked.Systems[system] == nil {
		locked.Systems[system] = &SystemInfo{}
	}
	locked.Systems[system].URL = url
	locked.Systems[system].Sha256 = sha256
	return f.Save()
}
