package lock

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"time"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl/devopt"
)

func SyncLockfiles(opts devopt.UpdateOpts) error {
	lockfilePaths, err := collectLockfiles()
	if err != nil {
		return err
	}

	preferredPackages, err := latestPackages(lockfilePaths)
	if err != nil {
		return err
	}

	if opts.ReferenceLockFilePath != "" {
		var referenceLockFile File
		if err := cuecfg.ParseFile(opts.ReferenceLockFilePath, &referenceLockFile); err != nil {
			return err
		}
		for key, pkg := range referenceLockFile.Packages {
			preferredPackages[key] = pkg
		}
	}

	for _, lockfilePath := range lockfilePaths {
		var lockFile File
		if err := cuecfg.ParseFile(lockfilePath, &lockFile); err != nil {
			return err
		}

		changed := false
		for key, preferredPkg := range preferredPackages {
			if pkg, exists := lockFile.Packages[key]; exists {
				if pkg.LastModified != preferredPkg.LastModified {
					lockFile.Packages[key].AllowInsecure = preferredPkg.AllowInsecure
					lockFile.Packages[key].LastModified = preferredPkg.LastModified
					// PluginVersion is intentionally omitted
					lockFile.Packages[key].Resolved = preferredPkg.Resolved
					lockFile.Packages[key].Source = preferredPkg.Source
					lockFile.Packages[key].Version = preferredPkg.Version
					lockFile.Packages[key].Systems = preferredPkg.Systems
					changed = true
				}
			}
		}

		if changed {
			if err = cuecfg.WriteFile(lockfilePath, lockFile); err != nil {
				return err
			}
			fmt.Printf("Updated: %s\n", lockfilePath)
		}
	}

	return nil
}

func latestPackages(lockfilePaths []string) (map[string]*Package, error) {
	latestPackages := make(map[string]*Package)

	for _, lockFilePath := range lockfilePaths {
		var lockFile File
		if err := cuecfg.ParseFile(lockFilePath, &lockFile); err != nil {
			return nil, err
		}
		for key, pkg := range lockFile.Packages {
			if latestPkg, exists := latestPackages[key]; exists {
				// Ignore error, which makes currentTime.After always false.
				currentTime, _ := time.Parse(time.RFC3339, pkg.LastModified)
				latestTime, err := time.Parse(time.RFC3339, latestPkg.LastModified)
				if err != nil {
					return nil, err
				}
				if currentTime.After(latestTime) {
					latestPackages[key] = pkg
				}
			} else if _, err := time.Parse(time.RFC3339, pkg.LastModified); err == nil {
				latestPackages[key] = pkg
			}
		}
	}

	return latestPackages, nil
}

func collectLockfiles() ([]string, error) {
	defer debug.FunctionTimer().End()

	var lockfiles []string
	err := filepath.WalkDir(
		".",
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !dirEntry.IsDir() && filepath.Base(path) == "devbox.lock" {
				lockfiles = append(lockfiles, path)
			}

			return nil
		},
	)

	return lockfiles, err
}
