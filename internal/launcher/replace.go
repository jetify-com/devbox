// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package launcher

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
)

// sudoCommand is a variable so that tests can replace it.
var sudoCommand = "sudo"

// ReplaceLauncher replaces the launcher at dst, which is either the bash
// launcher or a devbox binary, with the devbox binary at src. It runs sudo
// when it can't write to dst's directory, which is common for /usr/local/bin.
func ReplaceLauncher(src, dst string, stderr io.Writer) error {
	// Replace the file that a symlink points to, not the symlink.
	if resolved, err := filepath.EvalSymlinks(dst); err == nil {
		dst = resolved
	}

	err := replaceFile(src, dst)
	if err == nil || !errors.Is(err, fs.ErrPermission) {
		return err
	}

	sudo, lookErr := exec.LookPath(sudoCommand)
	if lookErr != nil {
		return fmt.Errorf("%w (and sudo isn't available)", err)
	}
	fmt.Fprintf(stderr, "Replacing the devbox launcher at %s requires sudo.\n", dst)
	cmd := exec.Command(
		sudo, "sh", "-c", `cp "$1" "$2.tmp" && chmod 755 "$2.tmp" && mv -f "$2.tmp" "$2"`,
		"sh", src, dst,
	)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("replace %s with sudo: %w", dst, err)
	}
	return nil
}

// replaceFile copies src to a temporary file next to dst and renames it to
// dst, so that commands that start at the same time never see a partial
// file.
func replaceFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	tmp, err := os.CreateTemp(filepath.Dir(dst), "."+filepath.Base(dst)+"-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	_, err = io.Copy(tmp, srcFile)
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o755); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), dst)
}
