// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"

	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/pkgdef"
	"go.jetify.com/devbox/internal/ux"
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
			// config to load. Warn instead, so that the definition's env
			// and init hook don't disappear silently.
			warnPkgDefConfig(pkg.Raw, err)
			continue
		}
		if config != nil {
			configs = append(configs, config)
		}
	}
	return configs, nil
}

// warnedPkgDefConfigs dedupes warnings: the config is loaded several times
// per command.
var warnedPkgDefConfigs sync.Map

// pkgDefWarningWriter is where activation errors are reported. Tests
// replace it.
var pkgDefWarningWriter io.Writer = os.Stderr

func warnPkgDefConfig(raw string, err error) {
	msg := fmt.Sprintf(
		"Skipping the env, init hook and create_files of %s: %v\n", raw, err,
	)
	if _, seen := warnedPkgDefConfigs.LoadOrStore(msg, true); seen {
		return
	}
	ux.Fwarningf(pkgDefWarningWriter, "%s", msg)
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

	system := nix.SystemOrHost()
	vars, err := def.TemplateVars(system, locked.Version)
	if err != nil {
		return nil, err
	}
	artifact := pkgdef.Artifact{}
	if info := locked.Systems[system]; info != nil {
		artifact = pkgdef.Artifact{URL: info.URL, Sha256: info.Sha256}
	}
	vars.InstallDir, err = client.InstallDir(def, locked.Version, system, artifact)
	if err != nil {
		return nil, err
	}
	vars.CacheDir = client.CacheDir

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

	content, err := renderPkgDefPluginContent(def, pkgDefTemplateData(includable, projectDir))
	if err != nil {
		return nil, fmt.Errorf("package definition %s: %w", def.Name, err)
	}
	jsonContent, err := json.Marshal(content)
	if err != nil {
		return nil, err
	}

	// The strings are already rendered, so this must not go through
	// buildConfig: its template pass would re-parse the JSON-escaped
	// content and fail on things like {{ if eq .OS "darwin" }}.
	cfg := &Config{PluginOnlyData: PluginOnlyData{Source: includable}}
	if err := json.Unmarshal(jsonContent, cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// pkgDefTemplateData is what a definition's activation templates see: the
// plugin.json variables ({{ .Virtenv }}, {{ .DevboxDir }}, ...) plus the
// definition's own ({{ .Version }}, {{ .InstallDir }}, ...).
func pkgDefTemplateData(p *pkgDefPlugin, projectDir string) map[string]string {
	data := pluginTemplateVars(p.CanonicalName(), projectDir)
	maps.Copy(data, p.vars)
	return data
}

// renderPkgDefPluginContent renders every string in the definition's env,
// shell and create_files (keys and values) with the package definition
// template engine, so that templates can use the definition's functions
// (trimPrefix, ...) and string literals.
func renderPkgDefPluginContent(def *pkgdef.Definition, data map[string]string) (*pkgDefPluginContent, error) {
	r := pkgDefRenderer{data: data, funcs: pkgdef.TemplateFuncs()}
	content := &pkgDefPluginContent{Name: def.Name, Description: def.Description}

	var err error
	if content.Env, err = r.stringMap(def.Env, false /*renderKeys*/); err != nil {
		return nil, fmt.Errorf("env: %w", err)
	}
	if content.CreateFiles, err = r.stringMap(def.CreateFiles, true /*renderKeys*/); err != nil {
		return nil, fmt.Errorf("create_files: %w", err)
	}
	if len(def.Shell) > 0 {
		content.Shell = make(map[string]json.RawMessage, len(def.Shell))
		for key, raw := range def.Shell {
			var value any
			if err := json.Unmarshal(raw, &value); err != nil {
				return nil, fmt.Errorf("shell.%s: %w", key, err)
			}
			rendered, err := r.value(value)
			if err != nil {
				return nil, fmt.Errorf("shell.%s: %w", key, err)
			}
			if content.Shell[key], err = json.Marshal(rendered); err != nil {
				return nil, err
			}
		}
	}
	return content, nil
}

type pkgDefRenderer struct {
	data  map[string]string
	funcs template.FuncMap
}

func (r pkgDefRenderer) render(s string) (string, error) {
	if !strings.Contains(s, "{{") {
		return s, nil
	}
	tmpl, err := template.New("pkgdef").
		Option("missingkey=error").
		Funcs(r.funcs).
		Parse(s)
	if err != nil {
		return "", fmt.Errorf("template %q: %w", s, err)
	}
	var buf strings.Builder
	if err := tmpl.Execute(&buf, r.data); err != nil {
		return "", fmt.Errorf("template %q: %w", s, err)
	}
	return buf.String(), nil
}

func (r pkgDefRenderer) stringMap(m map[string]string, renderKeys bool) (map[string]string, error) {
	if m == nil {
		return nil, nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		if renderKeys {
			var err error
			if k, err = r.render(k); err != nil {
				return nil, err
			}
		}
		rendered, err := r.render(v)
		if err != nil {
			return nil, err
		}
		out[k] = rendered
	}
	return out, nil
}

// value renders every string inside a decoded JSON value (init_hook is a
// string or a list, scripts a map of those).
func (r pkgDefRenderer) value(v any) (any, error) {
	switch v := v.(type) {
	case string:
		return r.render(v)
	case []any:
		for i := range v {
			rendered, err := r.value(v[i])
			if err != nil {
				return nil, err
			}
			v[i] = rendered
		}
		return v, nil
	case map[string]any:
		for k := range v {
			rendered, err := r.value(v[k])
			if err != nil {
				return nil, err
			}
			v[k] = rendered
		}
		return v, nil
	default:
		return v, nil
	}
}
