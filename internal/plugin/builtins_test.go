// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"go.jetify.com/devbox/plugins"
	"gopkg.in/yaml.v3"
)

// TestBuiltinPluginsAreValid loads every built-in plugin embedded in the
// binary the same way a project that adds its trigger package does, and checks
// that it would work. Built-in plugins are only exercised end to end by a few
// slow example projects (some of which only run nightly), so a typo in one of
// these files would otherwise ship and break `devbox add`/`shell`/`run` for
// every project that uses that package.
func TestBuiltinPluginsAreValid(t *testing.T) {
	entries, err := plugins.Builtins()
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) < 10 {
		t.Fatalf("found only %d built-in plugins, want the full set", len(entries))
	}

	builtin := &plugins.BuiltIn{}
	projectDir := filepath.Join(t.TempDir(), "project")
	for _, entry := range entries {
		file := entry.Name()
		if !strings.HasSuffix(file, ".json") {
			continue
		}
		name := strings.TrimSuffix(file, ".json")
		t.Run(name, func(t *testing.T) {
			content, err := builtin.FileContent(file)
			if err != nil {
				t.Fatal(err)
			}
			cfg, err := buildConfig(fakeIncludable{name: name}, projectDir, string(content))
			if err != nil {
				t.Fatalf("buildConfig(%s): %v", file, err)
			}
			if cfg.Name == "" {
				t.Error("plugin has no name")
			}
			if cfg.Version == "" {
				t.Error("plugin has no version; devbox.lock records it as plugin_version")
			}
			for env, value := range cfg.Env {
				if strings.Contains(value, "{{") {
					t.Errorf("env %s = %q still has a template action after rendering", env, value)
				}
			}

			for dest, src := range cfg.CreateFiles {
				// Plugins must only write inside the project.
				rel, err := filepath.Rel(projectDir, dest)
				if err != nil || !filepath.IsLocal(rel) {
					t.Errorf("create_files destination %q is outside the project dir %q", dest, projectDir)
				}
				if src == "" {
					continue // creates a directory
				}
				checkBuiltinFileTemplate(t, builtin, dest, src)
			}
		})
	}
}

// checkBuiltinFileTemplate checks that the create_files source src exists in
// the binary and renders with the data that Manager.createFile provides.
func checkBuiltinFileTemplate(t *testing.T, builtin *plugins.BuiltIn, dest, src string) {
	t.Helper()
	data, err := builtin.FileContent(src)
	if err != nil {
		t.Errorf("create_files %q -> %q: source isn't embedded in the binary: %v", dest, src, err)
		return
	}
	tmpl, err := template.New(src).Option("missingkey=error").Parse(string(data))
	if err != nil {
		t.Errorf("create_files source %q isn't a valid template: %v", src, err)
		return
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, map[string]any{
		"DevboxProjectDir":     "/project",
		"DevboxDir":            "/project/devbox.d/pkg",
		"DevboxDirRoot":        "/project/devbox.d",
		"DevboxProfileDefault": "/project/.devbox/nix/profile/default",
		"PackageAttributePath": "legacyPackages.x86_64-linux.pkg",
		"Packages":             []string{"pkg@latest"},
		"System":               "x86_64-linux",
		"URLForInput":          "github:NixOS/nixpkgs/0000000000000000000000000000000000000000",
		"Virtenv":              "/project/.devbox/virtenv/pkg",
	})
	if err != nil {
		t.Errorf("create_files source %q doesn't render: %v", src, err)
		return
	}
	// process-compose files define the plugin's services (devbox services
	// up), so they must still be valid YAML after rendering.
	if ext := filepath.Ext(src); ext == ".yaml" || ext == ".yml" {
		var v any
		if err := yaml.Unmarshal(buf.Bytes(), &v); err != nil {
			t.Errorf("create_files source %q isn't valid YAML after rendering: %v", src, err)
		}
	}
}
