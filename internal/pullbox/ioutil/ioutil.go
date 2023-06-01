package ioutil

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
)

func CopyAll(src, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, entry := range entries {
		cmd := exec.Command("cp", "-rf", filepath.Join(src, entry.Name()), dst)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func ClearDir(dir string) error {
	if err := os.RemoveAll(dir); err != nil {
		return errors.WithStack(err)
	}
	return os.MkdirAll(dir, 0755)
}
