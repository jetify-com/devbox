// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return errors.WithStack(err)
	}

	var srcFiles []fs.FileInfo
	if srcFileInfo.IsDir() {
		entries, err := os.ReadDir(src)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return errors.WithStack(err)
			}
			srcFiles = append(srcFiles, info)
		}
	} else {
		srcFiles = []fs.FileInfo{srcFileInfo}
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

	if overwrite {
		if err := os.RemoveAll(dst); err != nil {
			return errors.WithStack(err)
		}
		if err := os.MkdirAll(dst, 0755); err != nil {
			return errors.WithStack(err)
		}
	}

	for _, srcFile := range srcFiles {
		srcPath := src
		if srcFileInfo.IsDir() {
			srcPath = filepath.Join(src, srcFile.Name())
		}
		cmd := exec.Command("cp", "-rf", srcPath, dst)
		cmd.Stderr = os.Stderr
		cmd.Stdout = os.Stdout
		if err := cmd.Run(); err != nil {
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

// urlIsArchive checks if a file URL points to an archive file
func urlIsArchive(url string) (bool, error) {
	response, err := http.Head(url)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	contentType := response.Header.Get("Content-Type")
	return strings.Contains(contentType, "tar") ||
		strings.Contains(contentType, "zip") ||
		strings.Contains(contentType, "octet-stream"), nil
}
