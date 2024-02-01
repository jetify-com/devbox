// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/vercheck"
)

func ProfileUpgrade(ProfileDir, indexOrName string) error {
	cmd := command(
		"profile", "upgrade",
		"--profile", ProfileDir,
		indexOrName,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix profile upgrade\": %s: %w", out, err,
		)
	}
	return nil
}

func FlakeUpdate(ProfileDir string) error {
	version, err := Version()
	if err != nil {
		return err
	}
	ux.Finfo(os.Stderr, "Running \"nix flake update\"\n")
	cmd := exec.Command("nix", "flake", "update")
	if vercheck.SemverCompare(version, "2.19.0") >= 0 {
		cmd.Args = append(cmd.Args, "--flake")
	}
	cmd.Args = append(cmd.Args, ProfileDir)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix flake update\": %s: %w", out, err)
	}
	return nil
}
