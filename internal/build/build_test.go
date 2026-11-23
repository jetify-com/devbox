// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package build

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The release pipeline injects the version, commit and other metadata into the
// variables in this package with `-ldflags -X importpath.Name=value`. The Go
// linker silently ignores -X flags that name a variable that doesn't exist, or
// one that isn't a string initialized to a constant. If one of these variables
// is renamed, moved, or given a computed default, release builds would still
// succeed but report "0.0.0-dev" (and behave like a dev build), and nothing
// else in CI would notice. These tests pin the ldflags in .goreleaser.yaml and
// flake.nix to the variables declared here.

const importPath = "go.jetify.com/devbox/internal/build"

// ldflagX matches the target of a -X linker flag, such as
// "go.jetify.com/devbox/internal/build.Version" in
// "-X go.jetify.com/devbox/internal/build.Version={{.Version}}".
var ldflagX = regexp.MustCompile(`-X\s+([^\s=]+)\.([A-Za-z_][A-Za-z0-9_]*)=`)

func TestGoreleaserLdflagsSetExistingVars(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Builds []struct {
			Main    string   `yaml:"main"`
			Ldflags []string `yaml:"ldflags"`
		} `yaml:"builds"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse .goreleaser.yaml: %v", err)
	}
	if len(cfg.Builds) == 0 {
		t.Fatal(".goreleaser.yaml has no builds")
	}
	for _, b := range cfg.Builds {
		got := checkLdflags(t, ".goreleaser.yaml", strings.Join(b.Ldflags, " "))
		// Release builds must stamp the version and act as their own
		// launcher; without these, installed devbox reports 0.0.0-dev and
		// never switches versions.
		for _, want := range []string{"Version", "Commit", "CommitDate", "Launcher"} {
			if !slices.Contains(got, want) {
				t.Errorf(".goreleaser.yaml build %q doesn't set %s.%s with -X", b.Main, importPath, want)
			}
		}
	}
}

func TestFlakeLdflagsSetExistingVars(t *testing.T) {
	data, err := os.ReadFile(filepath.Join(repoRoot(t), "flake.nix"))
	if err != nil {
		t.Fatal(err)
	}
	got := checkLdflags(t, "flake.nix", string(data))
	for _, want := range []string{"Version", "Commit"} {
		if !slices.Contains(got, want) {
			t.Errorf("flake.nix doesn't set %s.%s with -X", importPath, want)
		}
	}
	// The flake must not turn on the launcher: package managers such as
	// nixpkgs stay in charge of which devbox version is installed.
	if slices.Contains(got, "Launcher") {
		t.Errorf("flake.nix sets %s.Launcher, but only official release builds should be launchers", importPath)
	}
}

// checkLdflags reports an error for every -X flag in flags that doesn't name a
// string variable in this package that the linker can set. It returns the
// names of the variables that flags set.
func checkLdflags(t *testing.T, source, flags string) []string {
	t.Helper()
	settable := settableStringVars(t)
	var names []string
	for _, m := range ldflagX.FindAllStringSubmatch(flags, -1) {
		pkg, name := m[1], m[2]
		if pkg != importPath {
			t.Errorf("%s sets -X %s.%s, but only %s is expected to be set at link time", source, pkg, name, importPath)
			continue
		}
		if !settable[name] {
			t.Errorf("%s sets -X %s.%s, but there's no package-level string variable %s "+
				"with a constant (or no) initializer in internal/build, so the linker would silently ignore it",
				source, pkg, name, name)
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		t.Errorf("found no -X %s.* linker flags in %s", importPath, source)
	}
	return names
}

// settableStringVars returns the package-level variables in this package that
// `go build -ldflags -X` can set: string variables that are uninitialized or
// initialized to a constant string.
func settableStringVars(t *testing.T) map[string]bool {
	t.Helper()
	fset := token.NewFileSet()
	matches, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	vars := map[string]bool{}
	for _, path := range matches {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, decl := range f.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.VAR {
				continue
			}
			for _, spec := range gen.Specs {
				valueSpec := spec.(*ast.ValueSpec)
				for i, name := range valueSpec.Names {
					if valueSpec.Type != nil {
						ident, ok := valueSpec.Type.(*ast.Ident)
						if !ok || ident.Name != "string" {
							continue
						}
						if len(valueSpec.Values) == 0 {
							vars[name.Name] = true
							continue
						}
					}
					if i < len(valueSpec.Values) {
						lit, ok := valueSpec.Values[i].(*ast.BasicLit)
						if ok && lit.Kind == token.STRING {
							vars[name.Name] = true
						}
					}
				}
			}
		}
	}
	return vars
}

func TestSettableStringVarsMatchesPackage(t *testing.T) {
	// Sanity check the parser above against the variables we know about, so
	// that a bug in it can't make the ldflags tests pass vacuously.
	vars := settableStringVars(t)
	for _, name := range []string{"Version", "Commit", "CommitDate", "SentryDSN", "TelemetryKey", "Launcher"} {
		if !vars[name] {
			t.Errorf("settableStringVars() is missing %s", name)
		}
	}
	if vars["IsDev"] {
		t.Error("settableStringVars() includes IsDev, which is a bool and can't be set with -X")
	}
}

func TestDefaultsAreDevBuild(t *testing.T) {
	// Tests run without ldflags, so they see the defaults. Other packages
	// rely on these defaults to detect dev builds (which skip telemetry,
	// update checks and the launcher).
	if Version != "0.0.0-dev" {
		t.Errorf("Version = %q, want the dev default %q", Version, "0.0.0-dev")
	}
	if Launcher == "true" {
		t.Error("Launcher defaults to true, want it off unless set by release ldflags")
	}
	if os.Getenv("DEVBOX_PROD") == "" && !IsDev {
		t.Error("IsDev = false for a build without ldflags")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	dir, err := SourceDir()
	if err != nil {
		t.Fatalf("find devbox source dir: %v", err)
	}
	return dir
}
