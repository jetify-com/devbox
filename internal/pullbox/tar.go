// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/pkg/errors"
)

var ErrFileExists = errors.New("file already exists")

// extract decompresses a tar file and saves it to the specified directory
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

func (p *pullbox) copy(action Action, src, dst string) error {
	srcFiles, err := os.ReadDir(src)
	if err != nil {
		return errors.WithStack(err)
	}
	for _, srcFile := range srcFiles {
		if _, err := os.Stat(filepath.Join(dst, srcFile.Name())); err == nil {
			if action == NoAction {
				return ErrFileExists
			}
		}
	}

	for _, srcFile := range srcFiles {
		srcPath := filepath.Join(src, srcFile.Name())
		dstPath := filepath.Join(dst, srcFile.Name())

		_, err := os.Stat(dstPath)
		// Special case for devbox.json and merge if needed.
		if err == nil && action == MergeAction && srcFile.Name() == "devbox.json" {
			fmt.Fprintf(os.Stderr, "Merging devbox.json\n")
			if err := p.merger(srcPath, dstPath); err != nil {
				return err
			}
			// Copy if overwrite or file doesn't exist.
		} else if action == OverwriteAction || err != nil {
			if err := exec.Command("cp", "-rf", srcPath, dst).Run(); err != nil {
				return err
			}
		} else {
			fmt.Fprintf(os.Stderr, "Conflict, not replacing %s\n", dstPath)
		}
	}
	return nil
}
