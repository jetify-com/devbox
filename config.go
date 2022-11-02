// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/boxcli/usererr"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/planner/plansdk"
)

// Config defines a devbox environment as JSON.
type Config struct {
	// Packages is the slice of Nix packages that devbox makes available in
	// its environment. Deliberately do not omitempty.
	Packages []string `cue:"[...string]" json:"packages"`
	// InstallStage defines the actions that should be taken when
	// installing language-specific libraries.
	InstallStage *Stage `json:"install_stage,omitempty"`
	// BuildStage defines the actions that should be taken when
	// compiling the application binary.
	BuildStage *Stage `json:"build_stage,omitempty"`
	// StartStage defines the actions that should be taken when
	// starting (running) the application.
	StartStage *Stage `json:"start_stage,omitempty"`

	// Shell configures the devbox shell environment.
	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook ConfigShellCmds `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`

	// Nixpkgs specifies the repository to pull packages from
	Nixpkgs struct {
		Version string `json:"version,omitempty"`
	} `json:"nixpkgs,omitempty"`
}

// This contains a subset of fields from plansdk.Stage
type Stage struct {
	Command string `cue:"string" json:"command"`
}

// ReadConfig reads a devbox config file.
func ReadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := cuecfg.ParseFile(path, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

func upgradeConfig(cfg *Config, absFilePath string) error {
	if cfg.Nixpkgs.Version == "" && featureflag.Get(featureflag.NixpkgVersion).Enabled() {
		// For now, we add the hardcoded value corresponding to the commit hash as of 2022-08-16 in:
		// `git ls-remote https://github.com/nixos/nixpkgs nixos-unstable`
		// In the near future, this will be changed to the commit-hash of the unstable tag in nixpkgs github repository
		const defaultCommitHash = "af9e00071d0971eb292fd5abef334e66eda3cb69"
		debug.Log("Missing nixpkgs.version from config, so adding the default value of %s", defaultCommitHash)

		cfg.Nixpkgs.Version = defaultCommitHash
		return WriteConfig(absFilePath, cfg)
	}
	return nil
}

// WriteConfig saves a devbox config file.
func WriteConfig(path string, cfg *Config) error {
	return cuecfg.WriteFile(path, cfg)
}

// Formats for marshalling and unmarshalling a series of shell commands in a
// devbox config.
const (
	// CmdArray formats shell commands as an array of of strings.
	CmdArray CmdFormat = iota

	// CmdString formats shell commands as a single string.
	CmdString
)

// CmdFormat defines a way of formatting shell commands in a devbox config.
type CmdFormat int

func (c CmdFormat) String() string {
	switch c {
	case CmdArray:
		return "array"
	case CmdString:
		return "string"
	default:
		return fmt.Sprintf("invalid (%d)", c)
	}
}

// ConfigShellCmds marshals and unmarshals shell commands from a devbox config
// as either a single string or an array of strings. It preserves the original
// value such that:
//
//	data == marshal(unmarshal(data)))
type ConfigShellCmds struct {
	// MarshalAs determines how MarshalJSON encodes the shell commands.
	// UnmarshalJSON will set MarshalAs automatically so that commands
	// marshal back to their original format. The default zero-value
	// formats them as an array.
	//
	MarshalAs CmdFormat
	Cmds      []string
}

// AppendScript appends each line of a script to s.Cmds. It also applies the
// following formatting rules:
//
//   - Trim leading newlines from the script.
//   - Trim trailing whitespace from the script.
//   - If the first line of the script begins with one or more tabs ('\t'), then
//     unindent each line by that same number of tabs.
//   - Remove trailing whitespace from each line.
//
// Note that unindenting only happens when a line starts with at least as many
// tabs as the first line. If it starts with fewer tabs, then it is not
// unindented at all.
func (s *ConfigShellCmds) AppendScript(script string) {
	script = strings.TrimLeft(script, "\r\n ")
	script = strings.TrimRightFunc(script, unicode.IsSpace)
	if len(script) == 0 {
		return
	}
	prefixLen := strings.IndexFunc(script, func(r rune) bool { return r != '\t' })
	prefix := strings.Repeat("\t", prefixLen)
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		line = strings.TrimPrefix(line, prefix)
		s.Cmds = append(s.Cmds, line)
	}
}

// MarshalJSON marshals shell commands according to s.MarshalAs. It marshals
// commands to a string by joining s.Cmds with newlines.
func (s ConfigShellCmds) MarshalJSON() ([]byte, error) {
	switch s.MarshalAs {
	case CmdArray:
		return json.Marshal(s.Cmds)
	case CmdString:
		return json.Marshal(s.String())
	default:
		panic(fmt.Sprintf("invalid command format: %s", s.MarshalAs))
	}
}

// UnmarshalJSON unmarshals shell commands from a string, an array of strings,
// or null. When the JSON value is a string, it unmarshals into the first index
// of s.Cmds.
func (s *ConfigShellCmds) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		s.MarshalAs = CmdArray
		s.Cmds = nil
		return nil
	}

	switch data[0] {
	case '"':
		s.MarshalAs = CmdString
		s.Cmds = []string{""}
		return json.Unmarshal(data, &s.Cmds[0])

	case '[':
		s.MarshalAs = CmdArray
		return json.Unmarshal(data, &s.Cmds)
	default:
		return nil
	}
}

// String formats the commands as a single string by joining them with newlines.
func (s *ConfigShellCmds) String() string {
	return strings.Join(s.Cmds, "\n")
}

// findConfigDir is a utility for using the path
func findConfigDir(path string) (string, error) {
	debug.Log("findConfigDir: path is %s\n", path)

	// Sanitize the directory and use the absolute path as canonical form
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// If the path  is specified, then we check directly for a config.
	// Otherwise, we search the parent directories.
	if path != "" {
		return findConfigDirAtPath(absPath)
	}
	return findConfigDirFromParentDirSearch("/" /*root*/, absPath)
}

func findConfigDirAtPath(absPath string) (string, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		if !plansdk.FileExists(filepath.Join(absPath, configFilename)) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		return absPath, nil
	default: // assumes 'file' i.e. mode.IsRegular()
		if !plansdk.FileExists(filepath.Clean(absPath)) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		// we return a directory from this function
		return filepath.Dir(absPath), nil
	}
}

func findConfigDirFromParentDirSearch(root string, absPath string) (string, error) {

	cur := absPath
	// Search parent directories for a devbox.json
	for cur != root {
		debug.Log("finding %s in dir: %s\n", configFilename, cur)
		if plansdk.FileExists(filepath.Join(cur, configFilename)) {
			return cur, nil
		}
		cur = filepath.Dir(cur)
	}
	if plansdk.FileExists(filepath.Join(cur, configFilename)) {
		return cur, nil
	}
	return "", missingConfigError(absPath, true /*didCheckParents*/)
}

func missingConfigError(path string, didCheckParents bool) error {

	var workingDir string
	wd, err := os.Getwd()
	if err == nil {
		workingDir = wd
	}
	// We try to prettify the `path` before printing
	if path == "." || path == "" || workingDir == path {
		path = "this directory"
	} else {
		// Instead of a long absolute directory, print the relative directory

		// if an error occurs, then just use `path`
		if workingDir != "" {
			relDir, err := filepath.Rel(workingDir, path)
			if err == nil {
				path = relDir
			}
		}
	}

	parentDirCheckAddendum := ""
	if didCheckParents {
		parentDirCheckAddendum = ", or any parent directories"
	}

	return usererr.New("No devbox.json found in %s%s. Did you run `devbox init` yet?", path, parentDirCheckAddendum)
}
