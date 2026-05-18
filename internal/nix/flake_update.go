// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"context"
	"os"

	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/nix"
)

func FlakeUpdate(ProfileDir string) error {
	ux.Finfof(os.Stderr, "Running \"nix flake update\"\n")
	cmd := Command("flake", "update")
	if nix.AtLeast(Version2_19) {
		cmd.Args = append(cmd.Args, "--flake")
	}
	cmd.Args = append(cmd.Args, ProfileDir)
	return cmd.Run(context.TODO())
}
