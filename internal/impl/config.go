// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl/shellcmd"
	"go.jetpack.io/devbox/internal/planner/plansdk"
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
		InitHook shellcmd.Commands             `json:"init_hook,omitempty"`
		Scripts  map[string]*shellcmd.Commands `json:"scripts,omitempty"`
	} `json:"shell,omitempty"`

	// Nixpkgs specifies the repository to pull packages from
	Nixpkgs NixpkgsConfig `json:"nixpkgs,omitempty"`
}

type NixpkgsConfig struct {
	Commit string `json:"commit,omitempty"`
}

// This contains a subset of fields from plansdk.Stage
type Stage struct {
	Command string `cue:"string" json:"command"`
}

func readConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := cuecfg.ParseFile(path, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

// ReadConfig reads a devbox config file, and validates it.
func ReadConfig(path string) (*Config, error) {
	cfg, err := readConfig(path)
	if err != nil {
		return nil, err
	}
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	return cfg, err
}

func upgradeConfig(cfg *Config, absFilePath string) error {
	if cfg.Nixpkgs.Commit == "" {
		debug.Log("Missing nixpkgs.version from config, so adding the default value of %s",
			plansdk.DefaultNixpkgsCommit)

		cfg.Nixpkgs.Commit = plansdk.DefaultNixpkgsCommit
		return WriteConfig(absFilePath, cfg)
	}
	return nil
}

// WriteConfig saves a devbox config file.
func WriteConfig(path string, cfg *Config) error {
	err := validateConfig(cfg)
	if err != nil {
		return err
	}
	return cuecfg.WriteFile(path, cfg)
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

func validateConfig(cfg *Config) error {

	fns := [](func(cfg *Config) error){
		validateNixpkg,
		validateScripts,
	}

	for _, fn := range fns {
		if err := fn(cfg); err != nil {
			return err
		}
	}
	return nil
}
func validateScripts(cfg *Config) error {
	for k := range cfg.Shell.Scripts {
		if strings.TrimSpace(k) == "" {
			return errors.New("cannot have script with empty name in devbox.json")
		}
		if strings.TrimSpace(cfg.Shell.Scripts[k].String()) == "" {
			return errors.New("cannot have an empty script value in devbox.json")
		}
	}
	return nil
}

func validateNixpkg(cfg *Config) error {
	if cfg.Nixpkgs.Commit == "" {
		return nil
	}

	const commitLength = 40
	if len(cfg.Nixpkgs.Commit) != commitLength {
		return usererr.New(
			"Expected nixpkgs.commit to be of length %d but it has length %d",
			commitLength,
			len(cfg.Nixpkgs.Commit),
		)
	}
	return nil
}
