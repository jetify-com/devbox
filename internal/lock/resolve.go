// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/nix/flake"
	"golang.org/x/sync/errgroup"
)

// FetchResolvedPackage fetches a resolution but does not write it to the lock
// struct. This allows testing new versions of packages without writing to the
// lock. This is useful to avoid changing nixpkgs commit hashes when version has
// not changed. This can happen when doing `devbox update` and search has
// a newer hash than the lock file but same version. In that case we don't want
// to update because it would be slow and wasteful.
func (f *File) FetchResolvedPackage(pkg string) (*Package, error) {
	if pkgtype.IsFlake(pkg) {
		installable, err := flake.ParseInstallable(pkg)
		if err != nil {
			return nil, fmt.Errorf("package %q: %v", pkg, err)
		}
		installable.Ref, err = lockFlake(context.TODO(), installable.Ref)
		if err != nil {
			return nil, err
		}
		return &Package{
			Resolved: installable.String(),
		}, nil
	}

	name, version, _ := searcher.ParseVersionedPackage(pkg)
	if version == "" {
		return nil, usererr.New("No version specified for %q.", name)
	}

	if pkgtype.IsRunX(pkg) {
		ref, err := ResolveRunXPackage(context.TODO(), pkg)
		if err != nil {
			return nil, err
		}
		return &Package{
			Resolved: ref.String(),
			Version:  ref.Version,
		}, nil
	}
	if featureflag.ResolveV2.Enabled() {
		return resolveV2(context.TODO(), name, version)
	}

	packageVersion, err := searcher.Client().Resolve(name, version)
	if err != nil {
		return nil, errors.Wrapf(nix.ErrPackageNotFound, "%s@%s", name, version)
	}

	sysInfos, err := buildLockSystemInfos(packageVersion)
	if err != nil {
		return nil, err
	}
	packageInfo, err := selectForSystem(packageVersion.Systems)
	if err != nil {
		return nil, fmt.Errorf("no systems found for package %q", name)
	}

	if len(packageInfo.AttrPaths) == 0 {
		return nil, fmt.Errorf("no attr paths found for package %q", name)
	}

	return &Package{
		LastModified: time.Unix(int64(packageInfo.LastUpdated), 0).UTC().
			Format(time.RFC3339),
		Resolved: fmt.Sprintf(
			"github:NixOS/nixpkgs/%s#%s",
			packageInfo.CommitHash,
			packageInfo.AttrPaths[0],
		),
		Version: packageInfo.Version,
		Source:  devboxSearchSource,
		Systems: sysInfos,
	}, nil
}

func resolveV2(ctx context.Context, name, version string) (*Package, error) {
	resolved, err := searcher.Client().ResolveV2(ctx, name, version)
	if errors.Is(err, searcher.ErrNotFound) {
		return nil, redact.Errorf("%s@%s: %w", name, version, nix.ErrPackageNotFound)
	}
	if err != nil {
		return nil, err
	}

	// /v2/resolve never returns a success with no systems.
	sysPkg, _ := selectForSystem(resolved.Systems)
	pkg := &Package{
		LastModified: sysPkg.LastUpdated.Format(time.RFC3339),
		Resolved:     sysPkg.FlakeInstallable.String(),
		Source:       devboxSearchSource,
		Version:      resolved.Version,
		Systems:      make(map[string]*SystemInfo, len(resolved.Systems)),
	}
	for sys, info := range resolved.Systems {
		if len(info.Outputs) != 0 {
			outputs := make([]Output, len(info.Outputs))
			for i, out := range info.Outputs {
				outputs[i] = Output{
					Name:    out.Name,
					Path:    out.Path,
					Default: out.Default,
				}
			}
			storePath := ""
			if len(outputs) > 0 {
				// We pick the first output as the store path. Note, this is sub-optimal because
				// it may not include all the default outputs of the nix package, but is what older
				// Devbox used to do. And this code is for backwards-compatibility.
				//
				// Unlike /v2/resolve, the /v1/resolve endpoint does not return the store path. It
				// returns the commit hash and we run `nix store path-from-hash-part` to get the store path.
				// For some packages, this would return the store path of the first default output.
				//
				// For example, curl has default outputs `bin` and `man`. Previously, we would only install
				// the `bin` output as `v1/resolve`'s commit hash would match that. With /v2/resolve, we
				// install both outputs. So, team members on older Devbox will see just `bin` installed while
				// team members on newer Devbox will see both `bin` and `man` installed.
				storePath = outputs[0].Path
			}
			pkg.Systems[sys] = &SystemInfo{
				Outputs:   outputs,
				StorePath: storePath,
			}
		}
	}
	return pkg, nil
}

func selectForSystem[V any](systems map[string]V) (v V, err error) {
	if v, ok := systems[nix.System()]; ok {
		return v, nil
	}
	if v, ok := systems["x86_64-linux"]; ok {
		return v, nil
	}
	for _, v := range systems {
		return v, nil
	}
	return v, redact.Errorf("no systems found")
}

func buildLockSystemInfos(pkg *searcher.PackageVersion) (map[string]*SystemInfo, error) {
	// guard against missing search data
	systems := lo.PickBy(pkg.Systems, func(sysName string, sysInfo searcher.PackageInfo) bool {
		return sysInfo.StoreHash != "" && sysInfo.StoreName != ""
	})

	group, ctx := errgroup.WithContext(context.Background())

	var storePathLock sync.RWMutex
	sysStorePaths := map[string]string{}
	for _sysName, _sysInfo := range systems {
		sysName := _sysName // capture range variable
		sysInfo := _sysInfo // capture range variable

		group.Go(func() error {
			// We should use devpkg.BinaryCache here, but it'll cause a circular reference
			// Just hardcoding for now. Maybe we should move that to nix.DefaultBinaryCache?
			path, err := nix.StorePathFromHashPart(ctx, sysInfo.StoreHash, "https://cache.nixos.org")
			if err != nil {
				// Should we report this to sentry to collect data?
				slog.Error("failed to resolve store path", "system", sysName, "store_hash", sysInfo.StoreHash, "err", err)
				// Instead of erroring, we can just skip this package. It can install via the slow path.
				return nil
			}
			storePathLock.Lock()
			sysStorePaths[sysName] = path
			storePathLock.Unlock()
			return nil
		})
	}
	err := group.Wait()
	if err != nil {
		return nil, err
	}

	sysInfos := map[string]*SystemInfo{}
	for sysName, storePath := range sysStorePaths {
		sysInfo := &SystemInfo{
			StorePath: storePath,
		}
		sysInfo.addOutputFromLegacyStorePath()
		sysInfos[sysName] = sysInfo
	}
	return sysInfos, nil
}

func lockFlake(ctx context.Context, ref flake.Ref) (flake.Ref, error) {
	if ref.Locked() {
		return ref, nil
	}

	// Nix requires a NAR hash for GitHub flakes to be locked. A Devbox lock
	// file is a bit more lenient and only requires a revision so that we
	// don't need to download the nixpkgs source for cached packages. If the
	// search index is ever able to return the NAR hash then we can remove
	// this check.
	if ref.Type == flake.TypeGitHub && (ref.Rev != "") {
		return ref, nil
	}

	meta, err := nix.ResolveFlake(ctx, ref)
	if err != nil {
		return ref, err
	}
	return meta.Locked, nil
}
