package lock

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
)

func SyncLockfiles() error {
	lockfilePaths, err := collectLockfiles()
	if err != nil {
		return err
	}

	latestPackages, err := latestPackages(lockfilePaths)
	if err != nil {
		return err
	}

	for _, lockfilePath := range lockfilePaths {
		var lockFile File
		if err := cuecfg.ParseFile(lockfilePath, &lockFile); err != nil {
			return err
		}

		changed := false
		for key, latestPkg := range latestPackages {
			if pkg, exists := lockFile.Packages[key]; exists {
				if pkg.LastModified != latestPkg.LastModified {
					lockFile.Packages[key].AllowInsecure = latestPkg.AllowInsecure
					lockFile.Packages[key].LastModified = latestPkg.LastModified
					// PluginVersion is intentionally omitted
					lockFile.Packages[key].Resolved = latestPkg.Resolved
					lockFile.Packages[key].Source = latestPkg.Source
					lockFile.Packages[key].Version = latestPkg.Version
					lockFile.Packages[key].Systems = latestPkg.Systems
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

	type filterFuncStackEntry struct {
		filterFunc func(string) bool
		path       string
	}

	var filterFuncStack []filterFuncStackEntry

	var filterFunc = func(path string) bool {
		// Start at the top of the stack which has most specific gitignore.
		for i := len(filterFuncStack) - 1; i >= 0; i-- {
			if strings.HasPrefix(path, filterFuncStack[i].path) {
				// Any gitignore that is not a prefix can be popped off the stack
				// because WalkDir is depth first which means we're already in a
				// different branch
				filterFuncStack = filterFuncStack[:i+1]
				return filterFuncStack[i].filterFunc(path)
			}
		}
		return false
	}

	var lockfiles []string
	err := filepath.WalkDir(
		".",
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			filename := filepath.Base(path)

			if filename == ".gitignore" {
				ignoreMatcher, err := ignore.CompileIgnoreFile(path)
				if err != nil {
					return errors.WithStack(err)
				}
				// push a new filterFunc onto the stack
				filterFuncStack = append(filterFuncStack, filterFuncStackEntry{
					ignoreMatcher.MatchesPath,
					filepath.Dir(path),
				})
			}

			if filterFunc(path) {
				if dirEntry.IsDir() {
					return filepath.SkipDir
				}
			} else if !dirEntry.IsDir() && filename == "devbox.lock" {
				lockfiles = append(lockfiles, path)
			}

			return nil
		},
	)

	return lockfiles, err
}
