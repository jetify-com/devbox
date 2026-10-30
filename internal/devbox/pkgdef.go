// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"maps"
	"slices"
	"strings"

	"github.com/samber/lo"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/pkgdef"
	"go.jetify.com/devbox/internal/ux"
)

// installPkgDefPackages installs every package-definition package
// ("pkg:<ref>@<version>") that is not installed yet.
func (d *Devbox) installPkgDefPackages(ctx context.Context) error {
	_, err := d.pkgDefBinPaths(ctx, true /*recordDigest*/)
	return err
}

// pkgDefBinPaths returns the bin directories of every package-definition
// package, installing those that are missing. Installs are idempotent, so
// this is cheap once everything is in place and safe to call whenever the
// environment is computed.
//
// recordDigest controls whether a digest observed for a definition without
// a checksum source is written to devbox.lock. Only the install path does
// that; computing the environment must never modify the lockfile.
func (d *Devbox) pkgDefBinPaths(ctx context.Context, recordDigest bool) ([]string, error) {
	paths := []string{}
	for _, pkg := range lo.Filter(d.InstallablePackages(), devpkg.IsPkgDef) {
		installed, err := d.installPkgDef(ctx, pkg, recordDigest)
		if err != nil {
			return nil, err
		}
		paths = append(paths, installed.BinDirs...)
	}
	return paths, nil
}

func (d *Devbox) installPkgDef(ctx context.Context, pkg *devpkg.Package, recordDigest bool) (*pkgdef.Installed, error) {
	locked, err := d.lockfile.Resolve(pkg.Raw)
	if err != nil {
		return nil, err
	}
	ref, err := pkgdef.ParseRef(pkg.Raw)
	if err != nil {
		return nil, err
	}

	client := pkgdef.NewClient()
	client.Stderr = d.stderr
	loaded, err := client.Load(ctx, ref, d.projectDir, locked.DefinitionHash)
	if err != nil {
		return nil, err
	}
	def := loaded.Definition

	system := nix.System()
	if !def.SupportsSystem(system) {
		return nil, usererr.New(
			"Package %s does not support %s (it supports %s). "+
				"Run `devbox add %s --exclude-platform %s` to skip it on this machine.",
			pkg.Raw, system, strings.Join(def.Platforms, ", "), pkg.Raw, system,
		)
	}

	artifact := pkgdef.Artifact{}
	info := locked.Systems[system]
	if info == nil && len(locked.Systems) > 0 {
		// The lockfile lists the systems the vendor publishes this version
		// for, and this is not one of them.
		return nil, usererr.New(
			"Package %s resolved to %s@%s, which %s does not publish for %s (it is available for %s). "+
				"Pick another version, or run `devbox add %s --exclude-platform %s` to skip it on this machine.",
			pkg.Raw, def.Name, locked.Version, def.Name, system,
			strings.Join(slices.Sorted(maps.Keys(locked.Systems)), ", "), pkg.Raw, system,
		)
	}
	if info != nil {
		artifact = pkgdef.Artifact{URL: info.URL, Sha256: info.Sha256, Sha512: info.Sha512}
	}

	if !client.IsInstalled(def, locked.Version, system) {
		ux.Finfof(d.stderr, "Installing %s@%s from %s\n", def.Name, locked.Version, lo.Ternary(artifact.URL != "", artifact.URL, ref.Source))
		if def.UsesScripts() {
			ux.Fwarningf(d.stderr, "Package definition %s runs shell commands while installing.\n", ref.Source)
		}
		if !artifact.HasDigest() {
			ux.Fwarningf(d.stderr, "Package definition %s has no checksum for %s; the digest of this download will be recorded in devbox.lock.\n", ref.Source, system)
		}
	}

	installed, err := client.Install(ctx, def, locked.Version, system, artifact)
	if err != nil {
		return nil, err
	}

	// Trust on first use: record the digest we observed so every other
	// machine verifies against it.
	if recordDigest && !artifact.HasDigest() && installed.Sha256 != "" {
		if err := d.lockfile.SetArtifactForPackage(pkg.Raw, system, installed.URL, installed.Sha256); err != nil {
			return nil, err
		}
	}
	return installed, nil
}
