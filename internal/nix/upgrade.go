// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"os"

	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/nix"
)

func ProfileUpgrade(ProfileDir, indexOrName string) error {
	return Command(
		"profile", "upgrade",
		"--profile", ProfileDir,
		indexOrName,
	).Run(context.TODO())
}

func FlakeUpdate(ProfileDir string) error {
	ux.Finfof(os.Stderr, "Running \"nix flake update\"\n")
	cmd := Command("flake", "update")
	if nix.AtLeast(Version2_19) {
		cmd.Args = append(cmd.Args, "--flake")
	}
	cmd.Args = append(cmd.Args, ProfileDir)
	return cmd.Run(context.TODO())
}
