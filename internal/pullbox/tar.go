// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devconfig"
)

// extract decompresses a tar file and saves it to a tmp directory
func extract(data []byte) (string, error) {
	tempFile, err := os.CreateTemp("", "temp.tar.gz")
	if err != nil {
		return "", err
	}
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	_, err = tempFile.Write(data)
	if err != nil {
		return "", err
	}

	tempDir, err := os.MkdirTemp("", "temp")
	if err != nil {
		return "", err
	}

	cmd := exec.Command("tar", "-xf", tempFile.Name(), "-C", tempDir)

	if err = cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			waitStatus := exitErr.Sys().(syscall.WaitStatus)
			return "", fmt.Errorf(
				"tar extraction failed with exit code: %d",
				waitStatus.ExitStatus(),
			)
		}
		return "", err
	}

	return tempDir, nil
}

func (p *pullbox) copy(overwrite bool, src, dst string) error {
	srcFiles, err := os.ReadDir(src)
	if err != nil {
		return errors.WithStack(err)
	}

	if !overwrite {
		for _, srcFile := range srcFiles {
			dstPath := filepath.Join(dst, srcFile.Name())
			// Only show error if file exists and is a modified config
			if _, err := os.Stat(dstPath); err == nil && isModifiedConfig(dstPath) {
				return fs.ErrExist
			}
		}
	}

	for _, srcFile := range srcFiles {
		srcPath := filepath.Join(src, srcFile.Name())
		if err := exec.Command("cp", "-rf", srcPath, dst).Run(); err != nil {
			return err
		}
	}
	return nil
}

func isModifiedConfig(path string) bool {
	if filepath.Base(path) == devconfig.DefaultName {
		cfg, err := devconfig.Load(path)
		if err != nil {
			return false
		}
		return !cfg.Equals(devconfig.DefaultConfig())
	}
	return false
}
