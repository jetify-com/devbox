// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"go.jetpack.io/devbox/shell"
)

func Shell(path string) error {
	// nix-shell only runs bash, which isn't great if the user has a
	// different default shell. Here we try to detect what their current
	// shell is, and then `exec` it to replace the bash process inside
	// nix-shell.
	sh, err := shell.Detect()
	if err != nil {
		// Fall back to running the vanilla Nix bash shell.
		return runFallbackShell(path)
	}

	// Naively running the user's shell has two problems:
	//
	// 1. The shell will source the user's rc file and potentially reorder
	// the PATH. This is especially a problem with some shims that prepend
	// their own directories to the front of the PATH, replacing the
	// Nix-installed packages.
	// 2. If their shell is bash, we end up double-sourcing their ~/.bashrc.
	// Once when nix-shell launches bash, and again when we exec it.
	//
	// To workaround this, first we store the current (outside of devbox)
	// PATH in ORIGINAL_PATH. Then we run a "pure" nix-shell to prevent it
	// from sourcing their ~/.bashrc. From inside the nix-shell (but before
	// launching the user's preferred shell) we store the PATH again in
	// PURE_NIX_PATH. When we're finally in the user's preferred shell, we
	// can use these env vars to set the PATH so that Nix packages are up
	// front, and all of the other programs come after.
	//
	// ORIGINAL_PATH is set by sh.StartCommand.
	// PURE_NIX_PATH is set by the shell hook in shell.nix.tmpl.
	_ = sh.SetInit(`
# Update the $PATH so the user can keep using programs that live outside of Nix,
# but prefer anything installed by Nix.
export PATH="$PURE_NIX_PATH:$ORIGINAL_PATH"

# Prepend to the prompt to make it clear we're in a devbox shell.
export PS1="(devbox) $PS1"
`)

	cmd := exec.Command("nix-shell", path)
	cmd.Args = append(cmd.Args, "--pure", "--command", sh.ExecCommand())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func runFallbackShell(path string) error {
	cmd := exec.Command("nix-shell", path)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func Exec(path string, command []string) error {
	runCmd := strings.Join(command, " ")
	cmd := exec.Command("nix-shell", "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = path
	return cmd.Run()
}

func PkgExists(pkg string) bool {
	_, found := PkgInfo(pkg)
	return found
}

type Info struct {
	NixName string
	Name    string
	Version string
	System  string
}

func PkgInfo(pkg string) (*Info, bool) {
	buf := new(bytes.Buffer)
	attr := fmt.Sprintf("nixpkgs.%s", pkg)
	cmd := exec.Command("nix-env", "--json", "-qa", "-A", attr)
	cmd.Stdout = buf
	err := cmd.Run()
	if err != nil {
		// nix-env returns an error if the package name is invalid, for now assume
		// all errors are invalid packages.
		return nil, false /* not found */
	}
	pkgInfo := parseInfo(pkg, buf.Bytes())
	if pkgInfo == nil {
		return nil, false /* not found */
	}
	return pkgInfo, true /* found */
}

func parseInfo(pkg string, data []byte) *Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	if len(results) != 1 {
		panic(fmt.Sprintf("unexpected number of results: %d", len(results)))
	}
	for _, result := range results {
		pkgInfo := &Info{
			NixName: pkg,
			Name:    result["pname"].(string),
			Version: result["version"].(string),
			System:  result["system"].(string),
		}
		return pkgInfo
	}
	return nil
}
