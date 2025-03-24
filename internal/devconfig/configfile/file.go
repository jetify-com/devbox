// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package configfile

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/tailscale/hujson"
	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/cachehash"
	"go.jetify.com/devbox/internal/devbox/shellcmd"
	"go.jetify.com/devbox/internal/build"
	"github.com/hashicorp/go-version"
)

const (
	DefaultName = "devbox.json"
)

// ConfigFile defines a devbox environment as JSON.
type ConfigFile struct {
	// AbsRootPath is the absolute path to the devbox.json or plugin.json file
	// it will not be set for github plugins.
	AbsRootPath string `json:"-"`

	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`

	// Let's users specify the version of devbox.
	DevboxVersion string `json:"devbox_version,omitempty"`

	// PackagesMutator is the slice of Nix packages that devbox makes available in
	// its environment. Deliberately do not omitempty.
	PackagesMutator PackagesMutator `json:"packages"`

	// Env allows specifying env variables
	Env map[string]string `json:"env,omitempty"`

	// Only allows "envsec" for now
	EnvFrom string `json:"env_from,omitempty"`

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

	ast *configAST
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
	Command string `json:"command"`
}

func (c *ConfigFile) Bytes() []byte {
	b := c.ast.root.Pack()
	return bytes.ReplaceAll(b, []byte("\t"), []byte("  "))
}

func (c *ConfigFile) Hash() (string, error) {
	if c.ast == nil {
		return cachehash.JSON(c)
	}
	ast := c.ast.root.Clone()
	ast.Minimize()
	return cachehash.Bytes(ast.Pack()), nil
}

func (c *ConfigFile) Equals(other *ConfigFile) bool {
	hash1, _ := c.Hash()
	hash2, _ := other.Hash()
	return hash1 == hash2
}

func (c *ConfigFile) NixPkgsCommitHash() string {
	if c == nil || c.Nixpkgs == nil {
		return ""
	}
	return c.Nixpkgs.Commit
}

func (c *ConfigFile) InitHook() *shellcmd.Commands {
	if c == nil || c.Shell == nil || c.Shell.InitHook == nil {
		return &shellcmd.Commands{}
	}
	return c.Shell.InitHook
}

// SaveTo writes the config to a file.
func (c *ConfigFile) SaveTo(path string) error {
	finalPath := path
  if filepath.Base(path) != DefaultName {
      finalPath = filepath.Join(path, DefaultName)
  }

	//return os.WriteFile(filepath.Join(path, DefaultName), c.Bytes(), 0o644)
	return os.WriteFile(filepath.Join(finalPath), c.Bytes(), 0o644)
}

// TODO: Can we remove SaveTo and just use Save()?
func (c *ConfigFile) Save() error {
	return c.SaveTo(c.AbsRootPath)
}

// Get returns the package with the given versionedName
func (c *ConfigFile) GetPackage(versionedName string) (*Package, bool) {
	name, version := parseVersionedName(versionedName)
	i := c.PackagesMutator.index(name, version)
	if i == -1 {
		return nil, false
	}
	return &c.PackagesMutator.collection[i], true
}

// TopLevelPackages returns the packages in the config file, but not the included ones.
// Semi-awkwardly named to avoid confusion with the Packages method on Config.
func (c *ConfigFile) TopLevelPackages() []Package {
	return c.PackagesMutator.collection
}

func LoadBytes(b []byte) (*ConfigFile, error) {
	jsonb, err := hujson.Standardize(slices.Clone(b))
	if err != nil {
		return nil, err
	}

	ast, err := parseConfig(b)
	if err != nil {
		return nil, err
	}
	cfg := &ConfigFile{
		PackagesMutator: PackagesMutator{ast: ast},
		ast:             ast,
	}
	if err := json.Unmarshal(jsonb, cfg); err != nil {
		return nil, err
	}
	return cfg, validateConfig(cfg)
}

func validateConfig(cfg *ConfigFile) error {
	fns := []func(cfg *ConfigFile) error{
		ValidateNixpkg,
		validateScripts,
		ValidateDevboxVersion,
	}

	for _, fn := range fns {
		if err := fn(cfg); err != nil {
			return err
		}
	}
	return nil
}

var whitespace = regexp.MustCompile(`\s`)

func validateScripts(cfg *ConfigFile) error {
	scripts := cfg.Scripts()
	for k := range scripts {
		if strings.TrimSpace(k) == "" {
			return errors.New("cannot have script with empty name in devbox.json")
		}
		if whitespace.MatchString(k) {
			return errors.Errorf(
				"cannot have script name with whitespace in devbox.json: %s", k)
		}
		if strings.TrimSpace(scripts[k].String()) == "" {
			return errors.Errorf(
				"cannot have an empty script body in devbox.json: %s", k)
		}
	}
	return nil
}

func ValidateNixpkg(cfg *ConfigFile) error {
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

func ValidateDevboxVersion(cfg *ConfigFile) error {
    if cfg.DevboxVersion == "" {
    	return usererr.New("Missing devbox_version field in config, suggested value: \"~%s\",", build.Version)
    }

    // Use hashicorp/go-version for version constraint checking
    constraints, err := version.NewConstraint(cfg.DevboxVersion)
    if err != nil {
        return usererr.New("Invalid devbox_version constraint in config: %s", cfg.DevboxVersion)
    }

    currentVersion, err := version.NewVersion(build.Version)
    if err != nil {
        return usererr.New("Invalid current devbox version: %s", build.Version)
    }

    if !constraints.Check(currentVersion) {
        return usererr.New("Devbox version mismatch: project requires version %s but your running version is %s",
            cfg.DevboxVersion, build.Version)
    }

    return nil
}

// SetDevboxVersion sets the devbox_version field in the config
func (c *ConfigFile) SetDevboxVersion(version string) {
    c.DevboxVersion = version
    c.SetStringField("DevboxVersion", version)
}
