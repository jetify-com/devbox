package pkgcfg

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func Remove(rootDir string, pkgs []string) error {
	for _, pkg := range pkgs {
		if err := os.RemoveAll(filepath.Join(rootDir, VirtenvPath, pkg)); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func RemoveInvalidSymlinks(rootDir string) error {
	binPath := filepath.Join(rootDir, VirtenvBinPath)
	if _, err := os.Stat(binPath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	dirEntry, err := os.ReadDir(binPath)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range dirEntry {
		_, err := os.Stat(filepath.Join(rootDir, VirtenvPath, "bin", entry.Name()))
		if errors.Is(err, os.ErrNotExist) {
			os.Remove(filepath.Join(rootDir, VirtenvPath, "bin", entry.Name()))
		}
	}
	return nil
}
