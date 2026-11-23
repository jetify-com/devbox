// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/pkgdef"
	"go.jetify.com/devbox/nix/flake"
)

const pkgDefActivationTest = `{
  "schema": 1,
  "name": "hello",
  "platforms": ["aarch64-darwin", "x86_64-darwin", "aarch64-linux", "x86_64-linux"],
  "versions": {"type": "static", "versions": ["v1.2.0"]},
  "install": {"type": "binary", "url": "file://vendor/hello-{{.Version}}.sh"},
  "env": {
    "HELLO_OS": "{{ if eq .OS \"darwin\" }}mac{{ else }}other{{ end }}",
    "HELLO_VERSION": "{{ .Version | trimPrefix \"v\" }}",
    "HELLO_HOME": "{{ .InstallDir }}",
    "HELLO_VIRTENV": "{{ .Virtenv }}"
  },
  "shell": {
    "init_hook": ["echo \"{{ .Version | replace \".\" \"_\" }}\""],
    "scripts": {"where": "echo {{ .DevboxDir }}"}
  },
  "create_files": {
    "{{ .Virtenv }}/{{ .OS }}.txt": "files/{{ .OS }}.txt"
  }
}`

func TestRenderPkgDefPluginContent(t *testing.T) {
	def, err := pkgdef.Parse([]byte(pkgDefActivationTest))
	if err != nil {
		t.Fatal(err)
	}
	includable := &pkgDefPlugin{
		loaded: &pkgdef.Loaded{Definition: def},
		vars: map[string]string{
			"InstallDir": "/cache/hello/v1.2.0",
			"Version":    "v1.2.0",
			"System":     "aarch64-darwin",
			"OS":         "darwin",
			"Arch":       "arm64",
		},
	}
	content, err := renderPkgDefPluginContent(def, pkgDefTemplateData(includable, "/project"))
	if err != nil {
		t.Fatal(err)
	}

	wantEnv := map[string]string{
		"HELLO_OS":      "mac",
		"HELLO_VERSION": "1.2.0",
		"HELLO_HOME":    "/cache/hello/v1.2.0",
		"HELLO_VIRTENV": "/project/.devbox/virtenv/pkgdef-hello",
	}
	for k, want := range wantEnv {
		if got := content.Env[k]; got != want {
			t.Errorf("env %s = %q, want %q", k, got, want)
		}
	}
	if got, want := string(content.Shell["init_hook"]), `["echo \"v1_2_0\""]`; got != want {
		t.Errorf("init_hook = %s, want %s", got, want)
	}
	if got, want := string(content.Shell["scripts"]), `{"where":"echo /project/devbox.d/pkgdef-hello"}`; got != want {
		t.Errorf("scripts = %s, want %s", got, want)
	}
	if got := content.CreateFiles["/project/.devbox/virtenv/pkgdef-hello/darwin.txt"]; got != "files/darwin.txt" {
		t.Errorf("create_files = %v, want the destination and source rendered", content.CreateFiles)
	}

	// The rendered content must load as a plugin config without another
	// template pass mangling it.
	jsonContent, err := json.Marshal(content)
	if err != nil {
		t.Fatal(err)
	}
	cfg := &Config{}
	if err := json.Unmarshal(jsonContent, cfg); err != nil {
		t.Fatal(err)
	}
	if cfg.Env["HELLO_OS"] != "mac" {
		t.Errorf("config env HELLO_OS = %q, want %q", cfg.Env["HELLO_OS"], "mac")
	}
}

func TestRenderPkgDefPluginContentError(t *testing.T) {
	def, err := pkgdef.Parse([]byte(pkgDefActivationTest))
	if err != nil {
		t.Fatal(err)
	}
	def.Env = map[string]string{"BAD": "{{ .NoSuchVar }}"}
	_, err = renderPkgDefPluginContent(def, map[string]string{})
	if err == nil || !strings.Contains(err.Error(), "env") {
		t.Errorf("got error %v, want an env template error", err)
	}
}

type testProject struct{ dir string }

func (p testProject) ConfigHash() (string, error)    { return "", nil }
func (p testProject) NixConfigHash() (string, error) { return "", nil }
func (p testProject) Stdenv() flake.Ref              { return flake.Ref{} }
func (p testProject) ProjectDir() string             { return p.dir }
func (p testProject) AllPackageNamesIncludingRemovedTriggerPackages() []string {
	return nil
}

// TestGetPkgDefConfigsWarns checks that a definition whose activation can't
// be loaded is reported to the user instead of silently dropping its env,
// and that it doesn't fail the config load.
func TestGetPkgDefConfigsWarns(t *testing.T) {
	featureflag.PkgDefs.EnableForTest(t)
	dir := t.TempDir()
	bad := strings.Replace(pkgDefActivationTest, `"HELLO_HOME": "{{ .InstallDir }}"`, `"HELLO_HOME": "{{ .Nope }}"`, 1)
	if err := os.WriteFile(filepath.Join(dir, "hello.pkg.json"), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}

	lockfile, err := lock.GetFile(testProject{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	raw := "pkg:./hello.pkg.json@1"
	lockfile.Packages[raw] = &lock.Package{
		Resolved: "pkg:./hello.pkg.json@v1.2.0",
		Version:  "v1.2.0",
		Source:   pkgdef.Source,
	}

	var stderr bytes.Buffer
	pkgDefWarningWriter = &stderr
	t.Cleanup(func() { pkgDefWarningWriter = os.Stderr })

	configs, err := GetPkgDefConfigsForPackages(
		[]configfile.Package{configfile.NewVersionOnlyPackage(raw, "")}, lockfile,
	)
	if err != nil {
		t.Fatalf("config load must not fail: %v", err)
	}
	if len(configs) != 0 {
		t.Errorf("got %d configs, want 0", len(configs))
	}
	if !strings.Contains(stderr.String(), raw) || !strings.Contains(stderr.String(), "Nope") {
		t.Errorf("warning = %q, want it to name the package and the template error", stderr.String())
	}
}

// writePkgDefProject writes hello.pkg.json (with a create_files source) into
// each of subdirs and locks "pkg:./<subdir>/hello.pkg.json@1" for each.
func writePkgDefProject(t *testing.T, subdirs ...string) (string, *lock.File, []configfile.Package) {
	t.Helper()
	featureflag.PkgDefs.EnableForTest(t)
	dir := t.TempDir()
	lockfile, err := lock.GetFile(testProject{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	pkgs := []configfile.Package{}
	for _, sub := range subdirs {
		if err := os.MkdirAll(filepath.Join(dir, sub, "files"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, sub, "hello.pkg.json"), []byte(pkgDefActivationTest), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, osName := range []string{"darwin", "linux"} {
			if err := os.WriteFile(filepath.Join(dir, sub, "files", osName+".txt"), []byte("v1"), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		raw := "pkg:./" + sub + "/hello.pkg.json@1"
		lockfile.Packages[raw] = &lock.Package{
			Resolved: "pkg:./" + sub + "/hello.pkg.json@v1.2.0",
			Version:  "v1.2.0",
			Source:   pkgdef.Source,
		}
		pkgs = append(pkgs, configfile.NewVersionOnlyPackage(raw, ""))
	}
	return dir, lockfile, pkgs
}

func TestGetPkgDefConfigsSkipsDisabledPlugin(t *testing.T) {
	_, lockfile, pkgs := writePkgDefProject(t, "a")
	pkgs[0].DisablePlugin = true
	configs, err := GetPkgDefConfigsForPackages(pkgs, lockfile)
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 0 {
		t.Errorf("got %d configs, want 0 for a package with disable_plugin", len(configs))
	}
}

// TestGetPkgDefConfigsRejectsDuplicateNames checks that two definitions with
// the same name don't share (and overwrite) one plugin directory.
func TestGetPkgDefConfigsRejectsDuplicateNames(t *testing.T) {
	_, lockfile, pkgs := writePkgDefProject(t, "a", "b")
	var stderr bytes.Buffer
	pkgDefWarningWriter = &stderr
	t.Cleanup(func() { pkgDefWarningWriter = os.Stderr })

	configs, err := GetPkgDefConfigsForPackages(pkgs, lockfile)
	if err != nil {
		t.Fatal(err)
	}
	if len(configs) != 1 {
		t.Fatalf("got %d configs, want 1", len(configs))
	}
	if got := configs[0].Source.LockfileKey(); got != pkgs[0].VersionedName() {
		t.Errorf("kept %s, want the first package %s", got, pkgs[0].VersionedName())
	}
	if !strings.Contains(stderr.String(), pkgs[1].VersionedName()) || !strings.Contains(stderr.String(), "same name") {
		t.Errorf("warning = %q, want it to name the skipped package", stderr.String())
	}
}

// TestPkgDefConfigHashIncludesCreateFiles checks that editing a create_files
// source of a local definition changes the config hash.
func TestPkgDefConfigHashIncludesCreateFiles(t *testing.T) {
	dir, lockfile, pkgs := writePkgDefProject(t, "a")
	hash := func() string {
		configs, err := GetPkgDefConfigsForPackages(pkgs, lockfile)
		if err != nil || len(configs) != 1 {
			t.Fatalf("got %d configs, err %v", len(configs), err)
		}
		h, err := configs[0].Hash()
		if err != nil {
			t.Fatal(err)
		}
		return h
	}
	before := hash()
	for _, osName := range []string{"darwin", "linux"} {
		if err := os.WriteFile(filepath.Join(dir, "a", "files", osName+".txt"), []byte("v2"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if after := hash(); after == before {
		t.Error("config hash did not change after editing a create_files source")
	}
}
