// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/searcher"
	"golang.org/x/sync/errgroup"
)

// FetchResolvedPackage fetches a resolution but does not write it to the lock
// struct. This allows testing new versions of packages without writing to the
// lock. This is useful to avoid changing nixpkgs commit hashes when version has
// not changed. This can happen when doing `devbox update` and search has
// a newer hash than the lock file but same version. In that case we don't want
// to update because it would be slow and wasteful.
func (f *File) FetchResolvedPackage(pkg string) (*Package, error) {
	name, version, _ := searcher.ParseVersionedPackage(pkg)
	if version == "" {
		return nil, usererr.New("No version specified for %q.", name)
	}

	packageVersion, err := searcher.Client().Resolve(name, version)
	if err != nil {
		return nil, errors.Wrapf(nix.ErrPackageNotFound, "%s@%s", name, version)
	}

	sysInfos := map[string]*SystemInfo{}
	if featureflag.RemoveNixpkgs.Enabled() {
		sysInfos, err = buildLockSystemInfos(packageVersion)
		if err != nil {
			return nil, err
		}
	}
	packageInfo, err := selectForSystem(packageVersion)
	if err != nil {
		return nil, err
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

func selectForSystem(pkg *searcher.PackageVersion) (searcher.PackageInfo, error) {
	if pi, ok := pkg.Systems[nix.System()]; ok {
		return pi, nil
	}
	if pi, ok := pkg.Systems["x86_64-linux"]; ok {
		return pi, nil
	}
	for _, v := range pkg.Systems {
		return v, nil
	}
	return searcher.PackageInfo{}, fmt.Errorf("no systems found for package %q", pkg.Name)
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
				debug.Log(
					"Failed to resolve store path for %s with storeHash %s. Error is %s.\n",
					sysName,
					sysInfo.StoreHash,
					err,
				)
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
		sysInfos[sysName] = &SystemInfo{
			StorePath: storePath,
		}
	}
	return sysInfos, nil
}
