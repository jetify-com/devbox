// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package wrapnix

import (
	"bytes"
	"context"
	_ "embed"
	"io/fs"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
)

type devboxer interface {
	NixBins(ctx context.Context) ([]string, error)
	ShellEnvHash(ctx context.Context) (string, error)
	ShellEnvHashKey() string
	ProjectDir() string
}

//go:embed wrapper.sh.tmpl
var wrapper string
var wrapperTemplate = template.Must(template.New("wrapper").Parse(wrapper))

// CreateWrappers creates wrappers for all the executables in nix paths
func CreateWrappers(ctx context.Context, devbox devboxer) error {
	shellEnvHash, err := devbox.ShellEnvHash(ctx)
	if err != nil {
		return err
	}

	// Remove all old wrappers
	_ = os.RemoveAll(filepath.Join(devbox.ProjectDir(), plugin.WrapperPath))

	// Recreate the bin wrapper directory
	destPath := filepath.Join(wrapperBinPath(devbox))
	_ = os.MkdirAll(destPath, 0755)

	bashPath := cmdutil.GetPathOrDefault("bash", "/bin/bash")

	bins, err := devbox.NixBins(ctx)
	if err != nil {
		return err
	}
	// get absolute path of devbox binary that the launcher script invokes
	// to avoid causing an infinite loop when coreutils gets installed
	executablePath, err := os.Executable()
	if err != nil {
		return err
	}

	for _, bin := range bins {
		if err = createWrapper(&createWrapperArgs{
			devboxer:         devbox,
			BashPath:         bashPath,
			Command:          bin,
			ShellEnvHash:     shellEnvHash,
			DevboxBinaryPath: executablePath,
			destPath:         filepath.Join(destPath, filepath.Base(bin)),
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	return createSymlinksForSupportDirs(devbox.ProjectDir())
}

type createWrapperArgs struct {
	devboxer
	BashPath         string
	Command          string
	ShellEnvHash     string
	DevboxBinaryPath string
	destPath         string
}

func createWrapper(args *createWrapperArgs) error {
	buf := &bytes.Buffer{}
	if err := wrapperTemplate.Execute(buf, args); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.WriteFile(args.destPath, buf.Bytes(), 0755))
}

// createSymlinksForSupportDirs creates symlinks for the support dirs
// (etc, lib, share) in the virtenv. Some tools (like mariadb) expect
// these to be in a dir relative to the bin.
//
// TODO: this is not perfect. using the profile path will not take into account
// any special stuff we do in flake.nix. We should use the nix store directly,
// but that is a bit more complicated. Nix merges any support directories
// recursively, so we need to do the same.
// e.g. if go_1_19 and go_1_20 are installed, .devbox/nix/profile/default/share/go/api
// will contain the union of both. We need to do the same.
func createSymlinksForSupportDirs(projectDir string) error {
	profilePath := filepath.Join(projectDir, nix.ProfilePath)
	if _, err := os.Stat(profilePath); errors.Is(err, fs.ErrNotExist) {
		return nil
	}

	supportDirs, err := os.ReadDir(profilePath)
	if err != nil {
		return err
	}

	for _, dir := range supportDirs {
		// bin has wrappers and is not a symlink
		if dir.Name() == "bin" {
			continue
		}

		oldname := filepath.Join(projectDir, nix.ProfilePath, dir.Name())
		newname := filepath.Join(projectDir, plugin.WrapperPath, dir.Name())

		if err := os.Symlink(oldname, newname); err != nil {
			// ignore if the symlink already exists
			if errors.Is(err, os.ErrExist) {
				existing, readerr := os.Readlink(newname)
				if readerr != nil {
					return errors.WithStack(readerr)
				}
				if existing == oldname {
					continue
				}
				return errors.Errorf("symlink %s already exists and points to %s", newname, existing)

			}
			return err
		}
	}
	return nil
}

func wrapperBinPath(devbox devboxer) string {
	return filepath.Join(devbox.ProjectDir(), plugin.WrapperBinPath)
}
