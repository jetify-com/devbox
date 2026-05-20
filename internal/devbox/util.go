// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/pkg/errors"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/xdg"
)

const processComposeVersion = "1.110.0"

var utilProjectConfigPath string

func initDevboxUtilityProject(ctx context.Context, stderr io.Writer) error {
	devboxUtilityProjectPath, err := ensureDevboxUtilityConfig()
	if err != nil {
		return err
	}

	box, err := Open(&devopt.Opts{
		Dir:    devboxUtilityProjectPath,
		Stderr: stderr,
	})
	if err != nil {
		return errors.WithStack(err)
	}

	// Add all utilities here.
	utilities := []string{
		"process-compose@" + processComposeVersion,
	}

	// Skip Add for utilities whose exact versioned name is already in the
	// config; calling Add anyway would print noisy "Package already in
	// devbox.json" messages on every services interaction. A version mismatch
	// (e.g. after bumping processComposeVersion) will fall through to Add,
	// which replaces the existing package by canonical name.
	existing := box.AllPackageNamesIncludingRemovedTriggerPackages()
	toAdd := []string{}
	for _, u := range utilities {
		if !slices.Contains(existing, u) {
			toAdd = append(toAdd, u)
		}
	}
	if len(toAdd) > 0 {
		if err = box.Add(ctx, toAdd, devopt.AddOpts{}); err != nil {
			return err
		}
	}

	return box.Install(ctx)
}

func ensureDevboxUtilityConfig() (string, error) {
	if utilProjectConfigPath != "" {
		return utilProjectConfigPath, nil
	}

	path, err := utilityDataPath()
	if err != nil {
		return "", err
	}

	err = EnsureConfig(path)
	if err != nil {
		return "", err
	}

	// Avoids unnecessarily initializing the config again by caching the path
	utilProjectConfigPath = path

	return path, nil
}

func utilityLookPath(binName string) (string, error) {
	binPath, err := utilityBinPath()
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(binPath, binName)
	_, err = os.Stat(absPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return absPath, nil
}

func utilityDataPath() (string, error) {
	path := xdg.DataSubpath("devbox/util")
	return path, errors.WithStack(os.MkdirAll(path, 0o755))
}

func utilityNixProfilePath() (string, error) {
	path, err := utilityDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(path, ".devbox/nix/profile"), nil
}

func utilityBinPath() (string, error) {
	nixProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return "", err
	}

	return filepath.Join(nixProfilePath, "default/bin"), nil
}
