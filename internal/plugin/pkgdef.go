// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/pkgdef"
)

// pkgDefPlugin is the Includable behind a package definition's activation
// block (env, shell, create_files). Devbox turns every "pkg:" package into
// one of these so the plugin manager applies the definition's environment
// without any special casing.
type pkgDefPlugin struct {
	pkg    *devpkg.Package
	loaded *pkgdef.Loaded

	// vars are the definition's extra template variables (InstallDir,
	// Version, ...), applied to plugin.json and to create_files content.
	vars map[string]string
}

// CanonicalName namespaces the definition's name so that its virtenv and
// devbox.d directories never collide with a built-in plugin or nix package
// of the same name (a project may have both nix "go" and "pkg:./go.pkg.jsonc").
func (p *pkgDefPlugin) CanonicalName() string {
	return "pkgdef-" + p.loaded.Definition.Name
}

// FileContent reads a create_files source relative to the definition file.
// Remote definitions have no directory to read from.
func (p *pkgDefPlugin) FileContent(subpath string) ([]byte, error) {
	if p.loaded.Dir == "" {
		return nil, errors.New("create_files is only supported for local package definitions")
	}
	return os.ReadFile(filepath.Join(p.loaded.Dir, subpath))
}

func (p *pkgDefPlugin) Hash() string {
	return p.loaded.Hash
}

func (p *pkgDefPlugin) LockfileKey() string {
	return p.pkg.Raw
}

// GetPkgDefConfigsForPackages returns a plugin config for every package
// definition package in packages that has been resolved in the lockfile.
func GetPkgDefConfigsForPackages(
	packages []configfile.Package,
	lockfile *lock.File,
) ([]*Config, error) {
	configs := []*Config{}
	for _, pkg := range devpkg.PackagesFromConfig(packages, lockfile) {
		if !pkg.IsPkgDef() || !pkg.IsInstallable() {
			continue
		}
		config, err := getPkgDefConfig(pkg, lockfile.ProjectDir())
		if err != nil {
			// Loading the config must not depend on the definition being
			// reachable: `devbox update` and `devbox rm` are the commands
			// that fix a missing or changed definition, and they need the
			// config to load. The install step reports the error.
			slog.Debug("skipping package definition plugin", "pkg", pkg.Raw, "err", err)
			continue
		}
		if config != nil {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

// pkgDefPluginContent is the subset of a package definition that maps onto
// plugin.json.
type pkgDefPluginContent struct {
	Name        string                     `json:"name"`
	Description string                     `json:"description,omitempty"`
	Env         map[string]string          `json:"env,omitempty"`
	Shell       map[string]json.RawMessage `json:"shell,omitempty"`
	CreateFiles map[string]string          `json:"create_files,omitempty"`
}

// getPkgDefConfig builds the plugin config for a package definition. It
// returns nil when the package has no lockfile entry yet (for example while
// `devbox add` is still resolving it).
func getPkgDefConfig(pkg *devpkg.Package, projectDir string) (*Config, error) {
	locked := pkg.Locked()
	if locked == nil {
		return nil, nil
	}
	ref, err := pkgdef.ParseRef(pkg.Raw)
	if err != nil {
		return nil, err
	}

	client := pkgdef.NewClient()
	loaded, err := client.Load(context.TODO(), ref, projectDir, locked.DefinitionHash)
	if err != nil {
		return nil, err
	}
	def := loaded.Definition

	system := nix.System()
	vars, err := def.TemplateVars(system, locked.Version)
	if err != nil {
		return nil, err
	}
	vars.InstallDir = client.InstallDir(def.Name, locked.Version, system)
	vars.CacheDir = client.CacheDir

	content, err := json.Marshal(pkgDefPluginContent{
		Name:        def.Name,
		Description: def.Description,
		Env:         def.Env,
		Shell:       def.Shell,
		CreateFiles: def.CreateFiles,
	})
	if err != nil {
		return nil, err
	}

	includable := &pkgDefPlugin{
		pkg:    pkg,
		loaded: loaded,
		vars: map[string]string{
			"InstallDir": vars.InstallDir,
			"CacheDir":   vars.CacheDir,
			"Version":    vars.Version,
			"System":     vars.System,
			"OS":         vars.OS,
			"Arch":       vars.Arch,
		},
	}
	return buildConfigWithVars(includable, projectDir, string(content), includable.vars)
}
