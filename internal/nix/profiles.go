// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/redact"
)

func ProfileList(writer io.Writer, profilePath string) ([]string, error) {

	cmd := command("profile", "list", "--profile", profilePath)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)

	// We set stderr to a different output than stdout
	// to ensure error output is not mingled with the stdout output
	// that we need to parse.
	cmd.Stderr = writer

	out, err := cmd.StdoutPipe()
	if err != nil {
		return nil, redact.Errorf("error creating stdout pipe: %w", redact.Safe(err))
	}
	if err := cmd.Start(); err != nil {
		return nil, redact.Errorf("error starting \"nix profile list\" command: %w", err)
	}

	scanner := bufio.NewScanner(out)
	scanner.Split(bufio.ScanLines)

	lines := []string{}
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := cmd.Wait(); err != nil {
		return nil, redact.Errorf("error running \"nix profile list\": %w", err)
	}
	return lines, nil
}

func ProfileInstall(writer io.Writer, profilePath string, urlForInstall string) error {

	cmd := command(
		"profile", "install",
		"--profile", profilePath,
		"--impure", // for NIXPKGS_ALLOW_UNFREE
		// Using an arbitrary priority to avoid conflicts with other packages.
		// Note that this is not really the priority we care about, since we
		// use the flake.nix to specify the priority.
		"--priority", nextPriority(profilePath),
		urlForInstall,
	)
	cmd.Env = allowUnfreeEnv()

	// If nix profile install runs as tty, the output is much nicer. If we ever
	// need to change this to our own writers, consider that you may need
	// to implement your own nicer output. --print-build-logs flag may be useful.
	cmd.Stdin = os.Stdin
	cmd.Stdout = writer
	cmd.Stderr = writer

	return cmd.Run()
}

func ProfileRemove(profilePath string, indexes []string) error {

	cmd := command(
		append([]string{
			"profile", "remove",
			"--profile", profilePath,
			"--impure", // for NIXPKGS_ALLOW_UNFREE
		}, indexes...)...,
	)
	cmd.Env = allowUnfreeEnv()

	out, err := cmd.CombinedOutput()
	if err != nil {
		return redact.Errorf("error running \"nix profile remove\": %s: %w", out, err)
	}
	return nil
}

type manifest struct {
	Elements []struct {
		Priority int `json:"priority"`
	} `json:"elements"`
}

func readManifest(profilePath string) (manifest, error) {
	data, err := os.ReadFile(filepath.Join(profilePath, "manifest.json"))
	if errors.Is(err, fs.ErrNotExist) {
		return manifest{}, nil
	}
	if err != nil {
		return manifest{}, err
	}

	var m manifest
	return m, json.Unmarshal(data, &m)
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
