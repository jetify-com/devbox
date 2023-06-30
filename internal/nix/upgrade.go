// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"fmt"
	"os"
	"os/exec"

	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/ux"
)

func ProfileUpgrade(ProfileDir string, idx int) error {
	cmd := command(
		"profile", "upgrade",
		"--profile", ProfileDir,
		fmt.Sprintf("%d", idx),
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
	ux.Finfo(os.Stderr, "Running \"nix flake update\"\n")
	cmd := exec.Command("nix", "flake", "update", ProfileDir)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf(
			"error running \"nix flake update\": %s: %w", out, err)

	}
	return nil
}
