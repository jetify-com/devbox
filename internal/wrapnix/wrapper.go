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
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/plugin"
	"go.jetpack.io/devbox/internal/xdg"
)

// Avoid wrapping bash and sed to prevent accidentally creating a recursive loop
// We use DEVBOX_SYSTEM_BASH and DEVBOX_SYSTEM_SED so normally we won't use
// user versions, but we want to be extra careful.
// This also has minor performance benefits.
var dontWrap = map[string]bool{
	"bash": true,
	"sed":  true,
}

type CreateWrappersArgs struct {
	NixBins         []string
	ShellEnvHash    string
	ShellEnvHashKey string
	ProjectDir      string
}

//go:embed wrapper.sh.tmpl
var wrapper string
var wrapperTemplate = template.Must(template.New("wrapper").Parse(wrapper))

// devboxSymlinkDir is the directory that has the symlink to the devbox binary,
// which is used by the bin-wrappers
var devboxSymlinkDir = xdg.CacheSubpath(filepath.Join("devbox", "bin", "current"))

// CreateWrappers creates wrappers for all the executables in nix paths
func CreateWrappers(ctx context.Context, args CreateWrappersArgs) error {
	defer debug.FunctionTimer().End()

	// Remove all old wrappers
	_ = os.RemoveAll(filepath.Join(args.ProjectDir, plugin.WrapperPath))

	// Recreate the bin wrapper directory
	destPath := filepath.Join(WrapperBinPath(args.ProjectDir))
	_ = os.MkdirAll(destPath, 0o755)

	bashPath := cmdutil.GetPathOrDefault(os.Getenv("DEVBOX_SYSTEM_BASH"), "/bin/bash")
	sedPath := cmdutil.GetPathOrDefault(os.Getenv("DEVBOX_SYSTEM_SED"), "/usr/bin/sed")

	if err := CreateDevboxSymlinkIfPossible(); err != nil {
		return err
	}

	for _, bin := range args.NixBins {
		if dontWrap[filepath.Base(bin)] {
			continue
		}
		if err := createWrapper(&createWrapperArgs{
			WrapperBinPath:     destPath,
			CreateWrappersArgs: args,
			BashPath:           bashPath,
			SedPath:            sedPath,
			Command:            bin,
			DevboxSymlinkDir:   devboxSymlinkDir,
			destPath:           filepath.Join(destPath, filepath.Base(bin)),
		}); err != nil {
			return errors.WithStack(err)
		}
	}

	return createSymlinksForSupportDirs(args.ProjectDir)
}

// CreateDevboxSymlinkIfPossible creates a symlink to the devbox binary.
//
// Needed because:
//
//  1. The bin-wrappers cannot invoke devbox via the Launcher. The Launcher script
//     invokes some coreutils commands that may themselves be installed by devbox
//     and so be bin-wrappers. This causes an infinite loop.
//
//     So, the bin-wrappers need to directly invoke the devbox binary.
//
//  2. The devbox binary's path will change when devbox is updated. Hence
//     using absolute paths to the devbox binaries in the bin-wrappers
//     will result in bin-wrappers invoking older devbox binaries.
//
//     So, the bin-wrappers need to use a symlink to the latest devbox binary. This
//     symlink is updated when devbox is updated.
func CreateDevboxSymlinkIfPossible() error {
	// Get the symlink path; create the symlink directory if it doesn't exist.
	if err := fileutil.EnsureDirExists(devboxSymlinkDir, 0o755, false /*chmod*/); err != nil {
		return err
	}
	currentDevboxSymlinkPath := filepath.Join(devboxSymlinkDir, "devbox")

	// Get the path to the devbox binary.
	execPath, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}
	devboxBinaryPath, evalSymlinkErr := filepath.EvalSymlinks(execPath)
	// we check the error below, because we always want to remove the symlink

	// We will always re-create this symlink to ensure correctness.
	if err := os.Remove(currentDevboxSymlinkPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.WithStack(err)
	}

	if evalSymlinkErr != nil {
		// This may return an error due to symlink loops. But we don't stop the
		// process for this reason, so the bin-wrappers can still be created.
		//
		// Once the symlink loop is fixed, and the bin-wrappers
		// will start working without needing to be re-created.
		//
		// nolint:nilerr
		debug.Log("Error evaluating symlink: %v", evalSymlinkErr)
		return nil
	}

	// Don't return error if error is os.ErrExist to protect against race conditions.
	if err := os.Symlink(devboxBinaryPath, currentDevboxSymlinkPath); err != nil && !errors.Is(err, os.ErrExist) {
		return errors.WithStack(err)
	}

	return nil
}

type createWrapperArgs struct {
	CreateWrappersArgs
	BashPath         string
	SedPath          string
	Command          string
	destPath         string
	DevboxSymlinkDir string
	WrapperBinPath   string // This is the  directory where all bin wrappers live
}

func createWrapper(args *createWrapperArgs) error {
	buf := &bytes.Buffer{}
	if err := wrapperTemplate.Execute(buf, args); err != nil {
		return errors.WithStack(err)
	}

	return errors.WithStack(os.WriteFile(args.destPath, buf.Bytes(), 0o755))
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

func WrapperBinPath(projectDir string) string {
	return filepath.Join(projectDir, plugin.WrapperBinPath)
}
