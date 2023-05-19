// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/impl/shellcmd"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// Config defines a devbox environment as JSON.
type Config struct {
	// Packages is the slice of Nix packages that devbox makes available in
	// its environment. Deliberately do not omitempty.
	Packages []string `cue:"[...string]" json:"packages"`

	// Env allows specifying env variables
	Env map[string]string `json:"env,omitempty"`
	// Shell configures the devbox shell environment.
	Shell *shellConfig `json:"shell,omitempty"`
	// Nixpkgs specifies the repository to pull packages from
	// Deprecated: Versioned packages don't need this
	Nixpkgs *NixpkgsConfig `json:"nixpkgs,omitempty"`

	// Reserved to allow including other config files. Proposed format is:
	// path: for local files
	// https:// for remote files
	// plugin: for built-in plugins
	// This is a similar format to nix inputs
	Include []string `json:"include,omitempty"`
}

type shellConfig struct {
	// InitHook contains commands that will run at shell startup.
	InitHook *shellcmd.Commands            `json:"init_hook,omitempty"`
	Scripts  map[string]*shellcmd.Commands `json:"scripts,omitempty"`
}

type NixpkgsConfig struct {
	Commit string `json:"commit,omitempty"`
}

// Stage contains a subset of fields from plansdk.Stage
type Stage struct {
	Command string `cue:"string" json:"command"`
}

func defaultConfig() *Config {
	return &Config{
		Shell: &shellConfig{
			Scripts: map[string]*shellcmd.Commands{
				"test": {
					Cmds: []string{"echo \"Error: no test specified\" && exit 1"},
				},
			},
			InitHook: &shellcmd.Commands{
				Cmds: []string{
					"echo 'Welcome to devbox!'",
				},
			},
		},
	}
}

func (c *Config) Hash() (string, error) {
	return cuecfg.Hash(c)
}

func (c *Config) NixPkgsCommitHash() string {
	if c == nil || c.Nixpkgs == nil {
		return plansdk.DefaultNixpkgsCommit
	}
	return c.Nixpkgs.Commit
}

func (c *Config) Scripts() map[string]*shellcmd.Commands {
	if c == nil || c.Shell == nil {
		return nil
	}
	return c.Shell.Scripts
}

func (c *Config) InitHook() *shellcmd.Commands {
	if c == nil || c.Shell == nil {
		return nil
	}
	return c.Shell.InitHook
}

func readConfig(path string) (*Config, error) {
	cfg := &Config{}
	return cfg, errors.WithStack(cuecfg.ParseFile(path, cfg))
}

// ReadConfig reads a devbox config file, and validates it.
func ReadConfig(path string) (*Config, error) {
	cfg, err := readConfig(path)
	if err != nil {
		return nil, err
	}
	return cfg, validateConfig(cfg)
}

func readConfigFromURL(url *url.URL) (*Config, error) {
	res, err := http.Get(url.String())
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close()
	cfg := &Config{}
	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	ext := filepath.Ext(url.Path)
	if !cuecfg.IsSupportedExtension(ext) {
		ext = ".json"
	}
	return cfg, cuecfg.Unmarshal(data, ext, cfg)
}

// WriteConfig saves a devbox config file.
func WriteConfig(path string, cfg *Config) error {
	err := validateConfig(cfg)
	if err != nil {
		return err
	}
	return cuecfg.WriteFile(path, cfg)
}

// findProjectDir walks up the directory tree looking for a devbox.json
// and upon finding it, will return the directory-path.
//
// If it doesn't find any devbox.json, then an error is returned.
func findProjectDir(path string) (string, error) {
	debug.Log("findProjectDir: path is %s\n", path)

	// Sanitize the directory and use the absolute path as canonical form
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// If the path  is specified, then we check directly for a config.
	// Otherwise, we search the parent directories.
	if path != "" {
		return findProjectDirAtPath(absPath)
	}
	return findProjectDirFromParentDirSearch("/" /*root*/, absPath)
}

func findProjectDirAtPath(absPath string) (string, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		if !fileutil.Exists(filepath.Join(absPath, configFilename)) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		return absPath, nil
	default: // assumes 'file' i.e. mode.IsRegular()
		if !fileutil.Exists(filepath.Clean(absPath)) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		// we return a directory from this function
		return filepath.Dir(absPath), nil
	}
}

func findProjectDirFromParentDirSearch(root string, absPath string) (string, error) {
	cur := absPath
	// Search parent directories for a devbox.json
	for cur != root {
		debug.Log("finding %s in dir: %s\n", configFilename, cur)
		if fileutil.Exists(filepath.Join(cur, configFilename)) {
			return cur, nil
		}
		cur = filepath.Dir(cur)
	}
	if fileutil.Exists(filepath.Join(cur, configFilename)) {
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
	fns := []func(cfg *Config) error{
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

var whitespace = regexp.MustCompile(`\s`)

func validateScripts(cfg *Config) error {
	scripts := cfg.Scripts()
	for k := range scripts {
		if strings.TrimSpace(k) == "" {
			return errors.New("cannot have script with empty name in devbox.json")
		}
		if whitespace.MatchString(k) {
			return errors.Errorf("cannot have script name with whitespace in devbox.json: %s", k)
		}
		if strings.TrimSpace(scripts[k].String()) == "" {
			return errors.Errorf("cannot have an empty script body in devbox.json: %s", k)
		}
	}
	return nil
}

func validateNixpkg(cfg *Config) error {
	hash := cfg.NixPkgsCommitHash()
	if hash == "" {
		return nil
	}

	const commitLength = 40
	if len(hash) != commitLength {
		return usererr.New(
			"Expected nixpkgs.commit to be of length %d but it has length %d",
			commitLength,
			len(hash),
		)
	}
	return nil
}
