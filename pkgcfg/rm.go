package pkgcfg

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func Remove(rootDir string, pkgs []string) error {
	for _, pkg := range pkgs {
		if err := os.RemoveAll(filepath.Join(rootDir, confPath, pkg)); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func RemoveInvalidSymlinks(rootDir string) error {
	dirEntry, err := os.ReadDir(filepath.Join(rootDir, confPath, "bin"))
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range dirEntry {
		_, err := os.Stat(filepath.Join(rootDir, confPath, "bin", entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			os.Remove(filepath.Join(rootDir, confPath, "bin", entry.Name()))
		}
	}
	return nil
}
