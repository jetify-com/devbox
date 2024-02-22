// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/tailscale/hujson"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devbox/shellcmd"
)

const (
	defaultName      = "devbox.json"
	defaultTySONName = "devbox.tson"
)

const (
	jsonFormat = iota
	tsonFormat
)

// ConfigFile defines a devbox environment as JSON.
type ConfigFile struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Version     string `json:"version"`

	// PackagesMutator is the slice of Nix packages that devbox makes available in
	// its environment. Deliberately do not omitempty.
	PackagesMutator packagesMutator `json:"packages"`

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

	ast    *configAST
	format int
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

const DefaultInitHook = "echo 'Welcome to devbox!' > /dev/null"

func DefaultConfig() *ConfigFile {
	cfg, err := loadBytes([]byte(fmt.Sprintf(`{
  "$schema": "https://raw.githubusercontent.com/jetpack-io/devbox/main/.schema/devbox.schema.json",
  "packages": [],
  "shell": {
    "init_hook": [
      "%s"
    ],
    "scripts": {
      "test": [
        "echo \"Error: no test specified\" && exit 1"
      ]
    }
  }
}
`, DefaultInitHook)))
	if err != nil {
		panic("default devbox.json is invalid: " + err.Error())
	}
	return cfg
}

func (c *ConfigFile) Bytes() []byte {
	b := c.ast.root.Pack()
	return bytes.ReplaceAll(b, []byte("\t"), []byte("  "))
}

func (c *ConfigFile) Hash() (string, error) {
	ast := c.ast.root.Clone()
	ast.Minimize()
	return cachehash.Bytes(ast.Pack())
}

func (c *ConfigFile) Equals(other *ConfigFile) bool {
	hash1, _ := c.Hash()
	hash2, _ := other.Hash()
	return hash1 == hash2
}

func (c *ConfigFile) NixPkgsCommitHash() string {
	// The commit hash for nixpkgs-unstable on 2023-10-25 from status.nixos.org
	const DefaultNixpkgsCommit = "75a52265bda7fd25e06e3a67dee3f0354e73243c"

	if c == nil || c.Nixpkgs == nil || c.Nixpkgs.Commit == "" {
		return DefaultNixpkgsCommit
	}
	return c.Nixpkgs.Commit
}

func (c *ConfigFile) InitHook() *shellcmd.Commands {
	if c == nil || c.Shell == nil {
		return nil
	}
	return c.Shell.InitHook
}

// SaveTo writes the config to a file.
func (c *ConfigFile) SaveTo(path string) error {
	if c.format != jsonFormat {
		return errors.New("cannot save config to non-json format")
	}
	return os.WriteFile(filepath.Join(path, defaultName), c.Bytes(), 0o644)
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

func loadBytes(b []byte) (*ConfigFile, error) {
	jsonb, err := hujson.Standardize(slices.Clone(b))
	if err != nil {
		return nil, err
	}

	ast, err := parseConfig(b)
	if err != nil {
		return nil, err
	}
	cfg := &ConfigFile{
		PackagesMutator: packagesMutator{ast: ast},
		ast:             ast,
	}
	if err := json.Unmarshal(jsonb, cfg); err != nil {
		return nil, err
	}
	return cfg, validateConfig(cfg)
}

func LoadConfigFromURL(url string) (*ConfigFile, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return loadBytes(data)
}

func validateConfig(cfg *ConfigFile) error {
	fns := []func(cfg *ConfigFile) error{
		ValidateNixpkg,
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

func IsConfigName(name string) bool {
	return slices.Contains(ValidConfigNames(), name)
}

func ValidConfigNames() []string {
	names := []string{defaultName}
	if featureflag.TySON.Enabled() {
		names = append(names, defaultTySONName)
	}
	return names
}
