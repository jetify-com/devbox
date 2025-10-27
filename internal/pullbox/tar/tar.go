// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package tar

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/cmdutil"
	"go.jetify.com/devbox/internal/fileutil"
)

// extract decompresses a tar file and saves it to a tmp directory
func Extract(data []byte) (string, error) {
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

func Compress(dir string) (string, error) {
	tmpDir, err := fileutil.CreateDevboxTempDir()
	if err != nil {
		return "", err
	}
	target := filepath.Join(tmpDir, "archive.tar.gz")
	cmd := cmdutil.CommandTTY("tar", "-czf", target, ".")

	cmd.Dir = dir

	if err = cmd.Start(); err != nil {
		return "", errors.WithStack(err)
	}

	return target, nil
}
