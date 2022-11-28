// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
)

// ProfilePath contains the contents of the profile generated via `nix-env --profile ProfilePath <command>`
// Instead of using directory, prefer using the devbox.ProfilePath() function that ensures the directory exists.
const ProfilePath = ".devbox/nix/profile/default"

func PkgExists(nixpkgsCommit, pkg string) bool {
	_, found := PkgInfo(nixpkgsCommit, pkg)
	return found
}

type Info struct {
	NixName string
	Name    string
	Version string
	System  string
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s-%s", i.Name, i.Version, i.System)
}

func Exec(path string, command []string) error {
	runCmd := strings.Join(command, " ")
	cmd := exec.Command("nix-shell", path, "--run", runCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return errors.WithStack(cmd.Run())
}

func PkgInfo(nixpkgsCommit, pkg string) (*Info, bool) {
	buf := new(bytes.Buffer)
	exactPackage := fmt.Sprintf("nixpkgs/%s#%s", nixpkgsCommit, pkg)
	cmd := exec.Command("nix", "search", "--json", exactPackage)
	cmd.Args = appendExperimentalFeatures(cmd.Args, "nix-command", "flakes")
	cmd.Stdout = buf
	debug.Log("running command: %s\n", cmd.String())
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
	for nixpkgs, result := range results {
		pkgInfo := &Info{
			NixName: pkg,
			Name:    result["pname"].(string),
			Version: result["version"].(string),
		}

		reLegacyPackages := regexp.MustCompile(fmt.Sprintf("legacyPackages\\.(.*)\\.%s", pkg))
		if reLegacyPackages.Match([]byte(nixpkgs)) {
			matches := reLegacyPackages.FindStringSubmatch(nixpkgs)

			// we set 2 matches because the first match is for the whole string,
			// and the second match is for the capturing group
			if len(matches) != 2 {
				msg := fmt.Sprintf("expected 1 system match in regexp for %s but got %d matches: %v", nixpkgs,
					len(matches), matches)
				panic(msg) // TODO savil. bubble up the error
			}
			pkgInfo.System = matches[1]
		}

		return pkgInfo
	}
	return nil
}

func appendExperimentalFeatures(args []string, features ...string) []string {
	for _, f := range features {
		args = append(args, "--extra-experimental-features", f)
	}
	return args
}
