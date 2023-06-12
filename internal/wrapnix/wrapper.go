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
	"go.jetpack.io/devbox/internal/services"
)

type devboxer interface {
	NixBins(ctx context.Context) ([]string, error)
	ShellEnvHash(ctx context.Context) (string, error)
	ShellEnvHashKey() string
	ProjectDir() string
	Services() (services.Services, error)
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

	services, err := devbox.Services()
	if err != nil {
		return err
	}

	// Remove all old wrappers
	_ = os.RemoveAll(filepath.Join(devbox.ProjectDir(), plugin.WrapperPath))

	// Recreate the bin wrapper directory
	destPath := filepath.Join(devbox.ProjectDir(), plugin.WrapperBinPath)
	_ = os.MkdirAll(destPath, 0755)

	bashPath := cmdutil.GetPathOrDefault("bash", "/bin/bash")
	for _, service := range services {
		if err = createWrapper(&createWrapperArgs{
			devboxer:     devbox,
			BashPath:     bashPath,
			Command:      service.Start,
			Env:          service.Env,
			ShellEnvHash: shellEnvHash,
			destPath:     filepath.Join(destPath, service.StartName()),
		}); err != nil {
			return err
		}
		if err = createWrapper(&createWrapperArgs{
			devboxer:     devbox,
			BashPath:     bashPath,
			Command:      service.Stop,
			Env:          service.Env,
			ShellEnvHash: shellEnvHash,
			destPath:     filepath.Join(destPath, service.StopName()),
		}); err != nil {
			return err
		}
	}

	bins, err := devbox.NixBins(ctx)
	if err != nil {
		return err
	}

	for _, bin := range bins {
		if err = createWrapper(&createWrapperArgs{
			devboxer:     devbox,
			BashPath:     bashPath,
			Command:      bin,
			ShellEnvHash: shellEnvHash,
			destPath:     filepath.Join(destPath, filepath.Base(bin)),
		}); err != nil {
			return errors.WithStack(err)
		}
	}
	if err = createDevboxSymlink(devbox.ProjectDir()); err != nil {
		return err
	}

	return createSymlinksForSupportDirs(devbox.ProjectDir())
}

type createWrapperArgs struct {
	devboxer
	BashPath     string
	Command      string
	Env          map[string]string
	ShellEnvHash string

	destPath string
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

// Creates a symlink for devbox in .devbox/virtenv/.wrappers/bin
// so that devbox can be available inside a pure shell
func createDevboxSymlink(projectDir string) error {

	// Get absolute path for where devbox is called
	devboxPath, err := filepath.Abs(os.Args[0])
	if err != nil {
		return errors.Wrap(err, "failed to create devbox symlink. Devbox command won't be available inside the shell")
	}
	// Create a symlink between devbox in .wrappers/bin
	err = os.Symlink(devboxPath, filepath.Join(projectDir, plugin.WrapperBinPath, "devbox"))
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.Wrap(err, "failed to create devbox symlink. Devbox command won't be available inside the shell")
	}
	return nil
}
