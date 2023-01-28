// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
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

func PkgExists(nixpkgsCommit, pkg string) bool {
	_, found := PkgInfo(nixpkgsCommit, pkg)
	return found
}

type Info struct {
	NixName string
	Name    string
	Version string
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
	for _, result := range results {
		pkgInfo := &Info{
			NixName: pkg,
			Name:    result["pname"].(string),
			Version: result["version"].(string),
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
func PrintDevEnv(nixFilePath string) (*varsAndFuncs, error) {
	cmd := exec.Command("nix", "print-dev-env",
		"-f", nixFilePath,
		"--extra-experimental-features", "nix-command",
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", "nix-command flakes",
		"--json")
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
