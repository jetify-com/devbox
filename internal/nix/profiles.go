// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/redact"
)

func ProfileList(writer io.Writer, profilePath string, useJSON bool) (string, error) {
	cmd := Command("profile", "list", "--profile", profilePath)
	if useJSON {
		cmd.Args = append(cmd.Args, "--json")
	}
	out, err := cmd.Output(context.TODO())
	if err != nil {
		return "", redact.Errorf("error running \"nix profile list\": %w", err)
	}
	return string(out), nil
}

type ProfileInstallArgs struct {
	Installables []string
	ProfilePath  string
	Writer       io.Writer
}

var ErrPriorityConflict = errors.New("priority conflict")

func ProfileInstall(ctx context.Context, args *ProfileInstallArgs) error {
	defer debug.FunctionTimer().End()

	cmd := Command(
		"profile", "install",
		"--profile", args.ProfilePath,
		"--offline", // makes it faster. Package is already in store
		"--impure",  // for NIXPKGS_ALLOW_UNFREE
		// Using an arbitrary priority to avoid conflicts with other packages.
		// Note that this is not really the priority we care about, since we
		// use the flake.nix to specify the priority.
		"--priority", nextPriority(args.ProfilePath),
	)

	FixInstallableArgs(args.Installables)
	cmd.Args = appendArgs(cmd.Args, args.Installables)
	cmd.Env = allowUnfreeEnv(os.Environ())

	// We used to attach this function to stdout and in in order to get the more interactive output.
	// However, now we do the building in nix.Build, by the time we install in profile everything
	// should already be in the store. We need to capture the output so we can decide if a conflict
	// happened.
	out, err := cmd.CombinedOutput(ctx)
	if bytes.Contains(out, []byte("error: An existing package already provides the following file")) {
		return ErrPriorityConflict
	}
	return err
}

// ProfileRemove removes packages from a profile.
// WARNING, don't use indexes, they are not supported by nix 2.20+
func ProfileRemove(profilePath string, packageNames ...string) error {
	defer debug.FunctionTimer().End()
	cmd := Command(
		"profile", "remove",
		"--profile", profilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
	)

	FixInstallableArgs(packageNames)
	cmd.Args = appendArgs(cmd.Args, packageNames)
	cmd.Env = allowUnfreeEnv(allowInsecureEnv(os.Environ()))
	return cmd.Run(context.TODO())
}

type manifest struct {
	Elements []struct {
		Priority int
	}
}

func readManifest(profilePath string) (manifest, error) {
	data, err := os.ReadFile(filepath.Join(profilePath, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest{}, nil
	}
	if err != nil {
		return manifest{}, err
	}

	type manifestModern struct {
		Elements map[string]struct {
			Priority int `json:"priority"`
		} `json:"elements"`
	}
	var modernMani manifestModern
	if err := json.Unmarshal(data, &modernMani); err == nil {
		// Convert to the result format
		result := manifest{}
		for _, e := range modernMani.Elements {
			result.Elements = append(result.Elements, struct{ Priority int }{e.Priority})
		}
		return result, nil
	}

	type manifestLegacy struct {
		Elements []struct {
			Priority int `json:"priority"`
		} `json:"elements"`
	}
	var legacyMani manifestLegacy
	if err := json.Unmarshal(data, &legacyMani); err != nil {
		return manifest{}, err
	}

	// Convert to the result format
	result := manifest{}
	for _, e := range legacyMani.Elements {
		result.Elements = append(result.Elements, struct{ Priority int }{e.Priority})
	}
	return result, nil
}

const DefaultPriority = 5

func nextPriority(profilePath string) string {
	// error is ignored because it's ok if the file doesn't exist
	m, _ := readManifest(profilePath)
	max := DefaultPriority
	for _, e := range m.Elements {
		if e.Priority > max {
			max = e.Priority
		}
	}
	// Each subsequent package gets a lower priority. This matches how flake.nix
	// behaves
	return fmt.Sprintf("%d", max+1)
}
