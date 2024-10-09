// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc/v2"
	"github.com/hashicorp/go-envparse"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/xdg"
)

func nixLinks() []string {
	return []string{
		"/nix/var/nix/profiles/default/etc/profile.d/nix-daemon.sh",
		filepath.Join(os.Getenv(envir.Home), ".nix-profile/etc/profile.d/nix.sh"),
		// logic introduced in https://github.com/NixOS/nix/pull/5588/files
		xdg.StateSubpath("nix/profile/etc/profile.d/nix.sh"),
		xdg.StateSubpath("nix/profiles/profile/etc/profile.d/nix.sh"),
	}
}

func SourceNixEnv() error {
	// if command is not in path, the source the nix startup files and hopefully
	// the command will be found. (we should still check that nix is actually
	// installed before we get here)
	srcFile := ""
	for _, f := range nixLinks() {
		if _, err := os.Stat(f); err == nil {
			srcFile = f
			break
		}
	}

	if srcFile == "" {
		return usererr.New(
			"Unable to find nix startup file. If /nix directory exists it's " +
				"possible the installation did not complete successfully. Follow " +
				"instructions at https://nixos.org/download.html for manual install.",
		)
	}

	// Source the nix script that sets the environment, and print the environment
	// variables that it sets, in a way we can parse.

	// NOTE: Only use shell built-ins in this script so that we don't introduce
	// any dependencies on an external binary.
	script := heredoc.Docf(`
		. %s;
		echo PATH=$PATH;
		echo NIX_PROFILES=$NIX_PROFILES;
		echo NIX_SSL_CERT_FILE=$NIX_SSL_CERT_FILE;
		echo MANPATH=$MANPATH;
	`, srcFile)
	cmd := exec.Command(
		"/bin/sh",
		"-c",
		script,
	)

	bs, err := cmd.CombinedOutput()
	if err != nil {
		// When there's an error, the output is usually an error message that
		// was printed to stderr and that we want in the error for debugging.
		return errors.Wrap(err, string(bs))
	}

	envvars, err := envparse.Parse(bytes.NewReader(bs))
	if err != nil {
		return errors.Wrap(err, "failed to parse nix env vars")
	}

	for k, v := range envvars {
		if len(strings.TrimSpace(v)) > 0 {
			os.Setenv(k, v)
		}
	}

	return nil
}
