package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"go.jetpack.io/devbox/internal/cuecfg"
)

func SyncLockfiles() error {
	latestPackages, err := latestPackages()
	if err != nil {
		return err
	}

	// Step 2: Update the devbox.lock files
	return filepath.Walk(".", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.IsDir() && filepath.Base(path) == "devbox.lock" {
			var lockFile File
			if err := cuecfg.ParseFile(path, &lockFile); err != nil {
				return err
			}

			changed := false
			for key, latestPkg := range latestPackages {
				if pkg, exists := lockFile.Packages[key]; exists {
					if pkg.LastModified != latestPkg.LastModified {
						lockFile.Packages[key] = latestPkg
						changed = true
					}
				}
			}

			if changed {
				if err = cuecfg.WriteFile(path, lockFile); err != nil {
					return err
				}
				fmt.Printf("Updated: %s\n", path)
			}
		}
		return nil
	})
}

func latestPackages() (map[string]*Package, error) {
	latestPackages := make(map[string]*Package)

	err := filepath.Walk(".", func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !fi.IsDir() && filepath.Base(path) == "devbox.lock" {
			var lockFile File
			if err := cuecfg.ParseFile(path, &lockFile); err != nil {
				return err
			}

			for key, pkg := range lockFile.Packages {
				if latestPkg, exists := latestPackages[key]; exists {
					currentTime, err := time.Parse(time.RFC3339, pkg.LastModified)
					if err != nil {
						return err
					}
					latestTime, err := time.Parse(time.RFC3339, latestPkg.LastModified)
					if err != nil {
						return err
					}
					if currentTime.After(latestTime) {
						latestPackages[key] = pkg
					}
				} else {
					latestPackages[key] = pkg
				}
			}
		}
		return nil
	})
	return latestPackages, err
}
