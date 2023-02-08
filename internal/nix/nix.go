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

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// Instead of using directory, prefer using the devbox.ProfilePath() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

var ErrPackageNotFound = errors.New("package not found")
var ErrPackageNotInstalled = errors.New("package not installed")

func PkgExists(nixpkgsCommit, pkg string) bool {
	_, found := PkgInfo(nixpkgsCommit, pkg)
	return found
}

type Info struct {
	// attribute key is different in flakes vs legacy so we should only use it
	// if we know exactly which version we are using
	attributeKey string
	NixName      string
	Name         string
	Version      string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.Name, i.Version)
}

func Exec(path string, command []string, env []string) error {
	runCmd := strings.Join(command, " ")
	cmd := exec.Command("nix-shell", path, "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(DefaultEnv(), env...)
	return errors.WithStack(usererr.NewExecError(cmd.Run()))
}

func PkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	if featureflag.Flakes.Enabled() {
		return flakesPkgInfo(nixpkgsCommit, pkg)
	}
	return legacyPkgInfo(nixpkgsCommit, pkg)
}

func legacyPkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	info, err := plansdk.GetNixpkgsInfo(nixpkgsCommit)
	if err != nil {
		return nil, false
	}
	cmd := exec.Command("nix-env", "-qa", "-A", pkg, "-f", info.URL, "--json")
	return pkgInfo(cmd, pkg)
}

func flakesPkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	exactPackage := fmt.Sprintf("nixpkgs/%s#%s", nixpkgsCommit, pkg)
	if nixpkgsCommit == "" {
		exactPackage = fmt.Sprintf("nixpkgs#%s", pkg)
	}

	cmd := exec.Command("nix", "search",
		"--extra-experimental-features", "nix-command flakes",
		"--json", exactPackage)
	return pkgInfo(cmd, pkg)
}

func pkgInfo(cmd *exec.Cmd, pkg string) (*Info, bool) {
	cmd.Env = DefaultEnv()
	cmd.Stderr = os.Stderr
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil, false /* not found */
	}
	pkgInfo := parseInfo(pkg, out)
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
	for key, result := range results {
		pkgInfo := &Info{
			attributeKey: key,
			NixName:      pkg,
			Name:         result["pname"].(string),
			Version:      result["version"].(string),
		}

		return pkgInfo
	}
	return nil
}

func DefaultEnv() []string {
	return append(os.Environ(), "NIXPKGS_ALLOW_UNFREE=1")
}

type varsAndFuncs struct {
	Functions map[string]string   // the key is the name, the value is the body.
	Variables map[string]variable // the key is the name.
}
type variable struct {
	Type  string // valid types are var, exported, and array.
	Value any    // can be a string or an array of strings (iff type is array).
}

// PrintDevEnv calls `nix print-dev-env -f <path>` and returns its output. The output contains
// all the environment variables and bash functions required to create a nix shell.
func PrintDevEnv(nixShellFilePath, nixFlakesFilePath string) (*varsAndFuncs, error) {
	cmd := exec.Command("nix", "print-dev-env")
	if featureflag.Flakes.Enabled() {
		cmd.Args = append(cmd.Args, nixFlakesFilePath)
	} else {
		cmd.Args = append(cmd.Args, "-f", nixShellFilePath)
	}
	cmd.Args = append(cmd.Args,
		"--extra-experimental-features", "nix-command",
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", "nix-command flakes",
		"--json")
	debug.Log("Running print-dev-env cmd: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	var vaf varsAndFuncs
	err = json.Unmarshal(out, &vaf)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &vaf, nil
}

// ProfileInstall calls nix profile install with default profile
func ProfileInstall(nixpkgsCommit, pkg string) error {
	cmd := exec.Command("nix", "profile", "install",
		"nixpkgs/"+nixpkgsCommit+"#"+pkg,
		"--extra-experimental-features", "nix-command flakes",
	)
	cmd.Env = DefaultEnv()
	out, err := cmd.CombinedOutput()
	if bytes.Contains(out, []byte("does not provide attribute")) {
		return ErrPackageNotFound
	}

	return errors.WithStack(err)
}

func ProfileRemove(nixpkgsCommit, pkg string) error {
	info, found := flakesPkgInfo(nixpkgsCommit, pkg)
	if !found {
		return ErrPackageNotFound
	}
	cmd := exec.Command("nix", "profile", "remove",
		info.attributeKey,
		"--extra-experimental-features", "nix-command flakes",
	)
	cmd.Env = DefaultEnv()
	out, err := cmd.CombinedOutput()
	if bytes.Contains(out, []byte("does not match any packages")) {
		return ErrPackageNotInstalled
	}

	return errors.WithStack(err)
}
