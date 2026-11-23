// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"go.jetify.com/devbox/nix/flake"
)

// These tests cover reading and writing devbox.lock. Every devbox command that
// loads a project goes through this code, so a regression here either rewrites
// users' checked-in lockfiles on every run (churning their git history) or
// drops entries from them.

const testStdenv = "github:NixOS/nixpkgs/0000000000000000000000000000000000000000"

type lockTestProject struct {
	dir      string
	packages []string
}

func (p *lockTestProject) ConfigHash() (string, error)    { return "config", nil }
func (p *lockTestProject) NixConfigHash() (string, error) { return "nix", nil }
func (p *lockTestProject) ProjectDir() string             { return p.dir }

func (p *lockTestProject) Stdenv() flake.Ref {
	ref, err := flake.ParseRef(testStdenv)
	if err != nil {
		panic(err)
	}
	return ref
}

func (p *lockTestProject) AllPackageNamesIncludingRemovedTriggerPackages() []string {
	return p.packages
}

func writeLockfile(t *testing.T, dir, content string) string {
	t.Helper()
	path := filepath.Join(dir, "devbox.lock")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGetFileMissingLockfile(t *testing.T) {
	project := &lockTestProject{dir: t.TempDir()}
	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatalf("GetFile() with no devbox.lock: %v", err)
	}
	if lockfile.LockFileVersion != lockFileVersion {
		t.Errorf("LockFileVersion = %q, want %q", lockfile.LockFileVersion, lockFileVersion)
	}
	if len(lockfile.Packages) != 0 {
		t.Errorf("Packages = %v, want none", lockfile.Packages)
	}

	// Nothing changed, so saving must not create a lockfile.
	if err := lockfile.Save(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(project.dir, "devbox.lock")); !os.IsNotExist(err) {
		t.Errorf("Save() with no changes created devbox.lock (stat err = %v)", err)
	}
}

func TestGetFileInvalidLockfile(t *testing.T) {
	project := &lockTestProject{dir: t.TempDir()}
	writeLockfile(t, project.dir, `{"lockfile_version": "1", "packages": {`)
	if _, err := GetFile(project); err == nil {
		t.Error("GetFile() with a truncated devbox.lock returned no error")
	}
}

func TestSaveDoesNotRewriteUnchangedLockfile(t *testing.T) {
	project := &lockTestProject{dir: t.TempDir()}
	// Deliberately not formatted the way devbox writes lockfiles, so that a
	// rewrite would change the bytes.
	original := `{"lockfile_version":"1","packages":{"hello@latest":{"last_modified":"2024-01-01T00:00:00Z","resolved":"github:NixOS/nixpkgs/1111111111111111111111111111111111111111#hello","source":"devbox-search","version":"2.12.1","systems":{"x86_64-linux":{"outputs":[{"name":"out","path":"/nix/store/aaaa-hello-2.12.1","default":true}]}}}}}`
	path := writeLockfile(t, project.dir, original)

	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := lockfile.Save(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != original {
		t.Errorf("Save() rewrote an unchanged lockfile:\ngot:  %s\nwant: %s", got, original)
	}
}

func TestLegacyStorePathLockfile(t *testing.T) {
	project := &lockTestProject{dir: t.TempDir()}
	original := `{
  "lockfile_version": "1",
  "packages": {
    "curl@7.87.0": {
      "last_modified": "2023-02-26T03:47:33Z",
      "resolved": "github:NixOS/nixpkgs/9952d6bc395f5841262b006fbace8dd7e143b634#curl",
      "source": "devbox-search",
      "version": "7.87.0",
      "systems": {
        "x86_64-linux": {
          "store_path": "/nix/store/bbbb-curl-7.87.0"
        }
      }
    },
    "hello@latest": {
      "resolved": "github:NixOS/nixpkgs/1111111111111111111111111111111111111111#hello",
      "source": "devbox-search",
      "version": "2.12.1"
    }
  }
}
`
	path := writeLockfile(t, project.dir, original)

	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}

	// In memory, the legacy store_path is presented as a default "out" output.
	sys := lockfile.Get("curl@7.87.0").Systems["x86_64-linux"]
	want := []Output{{Name: "out", Path: "/nix/store/bbbb-curl-7.87.0", Default: true}}
	if !slices.Equal(sys.Outputs, want) {
		t.Errorf("legacy store_path outputs = %+v, want %+v", sys.Outputs, want)
	}

	// Converting the legacy format in memory alone mustn't rewrite the file.
	if err := lockfile.Save(); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, path); got != original {
		t.Errorf("Save() rewrote a legacy lockfile that didn't change:\ngot:\n%s\nwant:\n%s", got, original)
	}

	// A real change writes the file, but keeps the legacy entry in its legacy
	// format rather than upgrading it behind the user's back.
	if err := lockfile.Remove("hello@latest"); err != nil {
		t.Fatal(err)
	}
	var onDisk struct {
		Packages map[string]struct {
			Systems map[string]map[string]any `json:"systems"`
		} `json:"packages"`
	}
	if err := json.Unmarshal([]byte(readFile(t, path)), &onDisk); err != nil {
		t.Fatal(err)
	}
	if _, ok := onDisk.Packages["hello@latest"]; ok {
		t.Error("Remove() didn't remove hello@latest from devbox.lock")
	}
	curlSys := onDisk.Packages["curl@7.87.0"].Systems["x86_64-linux"]
	if curlSys["store_path"] != "/nix/store/bbbb-curl-7.87.0" {
		t.Errorf("legacy entry lost its store_path after Save(): %v", curlSys)
	}
	if _, ok := curlSys["outputs"]; ok {
		t.Errorf("legacy entry was rewritten with outputs after Save(): %v", curlSys)
	}

	// And the in-memory copy still has the outputs after saving.
	sys = lockfile.Get("curl@7.87.0").Systems["x86_64-linux"]
	if !slices.Equal(sys.Outputs, want) {
		t.Errorf("outputs after Save() = %+v, want %+v", sys.Outputs, want)
	}
}

func TestTidyKeepsConfigPackagesAndStdenv(t *testing.T) {
	project := &lockTestProject{
		dir:      t.TempDir(),
		packages: []string{"hello@latest", "go@1.20"},
	}
	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"hello@latest", "go@1.20", "go@1.19", "github:foo/bar", testStdenv} {
		lockfile.Packages[name] = &Package{Resolved: "github:NixOS/nixpkgs/1111111111111111111111111111111111111111#" + name}
	}

	lockfile.Tidy()

	var got []string
	for name := range lockfile.Packages {
		got = append(got, name)
	}
	slices.Sort(got)
	want := []string{"github:NixOS/nixpkgs/0000000000000000000000000000000000000000", "go@1.20", "hello@latest"}
	if !slices.Equal(got, want) {
		t.Errorf("packages after Tidy() = %v, want %v", got, want)
	}
}

func TestSaveRoundTrip(t *testing.T) {
	project := &lockTestProject{dir: t.TempDir()}
	lockfile, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	lockfile.Packages["hello@latest"] = &Package{
		AllowInsecure: true,
		LastModified:  "2024-01-01T00:00:00Z",
		PluginVersion: "0.0.2",
		Resolved:      "github:NixOS/nixpkgs/1111111111111111111111111111111111111111#hello",
		Source:        devboxSearchSource,
		Version:       "2.12.1",
		Systems: map[string]*SystemInfo{
			"aarch64-darwin": {Outputs: []Output{
				{Name: "out", Path: "/nix/store/cccc-hello", Default: true},
				{Name: "man", Path: "/nix/store/dddd-hello-man"},
			}},
		},
	}
	if err := lockfile.Save(); err != nil {
		t.Fatal(err)
	}

	reread, err := GetFile(project)
	if err != nil {
		t.Fatal(err)
	}
	if reread.LockFileVersion != lockFileVersion {
		t.Errorf("LockFileVersion = %q, want %q", reread.LockFileVersion, lockFileVersion)
	}
	got, want := reread.Get("hello@latest"), lockfile.Get("hello@latest")
	if got == nil {
		t.Fatal("hello@latest missing after round trip")
	}
	if got.AllowInsecure != want.AllowInsecure || got.LastModified != want.LastModified ||
		got.PluginVersion != want.PluginVersion || got.Resolved != want.Resolved ||
		got.Source != want.Source || got.Version != want.Version {
		t.Errorf("round trip changed the entry:\ngot:  %+v\nwant: %+v", got, want)
	}
	if !got.Systems["aarch64-darwin"].Equals(want.Systems["aarch64-darwin"]) {
		t.Errorf("round trip changed systems:\ngot:  %v\nwant: %v", got.Systems, want.Systems)
	}
	if !reread.HasAllowInsecurePackages() {
		t.Error("HasAllowInsecurePackages() = false after round trip")
	}

	// Saving the re-read file again is a no-op.
	before := readFile(t, lockFilePath(project.dir))
	if err := reread.Save(); err != nil {
		t.Fatal(err)
	}
	if after := readFile(t, lockFilePath(project.dir)); after != before {
		t.Errorf("second Save() changed devbox.lock:\nbefore:\n%s\nafter:\n%s", before, after)
	}
}

func TestGetSkipsUnresolvedEntries(t *testing.T) {
	lockfile := &File{Packages: map[string]*Package{
		"resolved@1":   {Resolved: "github:NixOS/nixpkgs/1111111111111111111111111111111111111111#resolved"},
		"unresolved@1": {Version: "1"},
	}}
	if lockfile.Get("resolved@1") == nil {
		t.Error("Get() of a resolved entry = nil")
	}
	// An entry without "resolved" must be re-resolved, not used as is.
	if got := lockfile.Get("unresolved@1"); got != nil {
		t.Errorf("Get() of an unresolved entry = %+v, want nil", got)
	}
	if got := lockfile.Get("missing@1"); got != nil {
		t.Errorf("Get() of a missing entry = %+v, want nil", got)
	}
}

func TestIsLegacyPackage(t *testing.T) {
	tests := map[string]bool{
		"hello":                              true,
		"hello@latest":                       false,
		"python@3.12":                        false,
		"github:NixOS/nixpkgs#hello":         false,
		"path:./my-flake#pkg":                false,
		"runx:golangci/golangci-lint@latest": false,
		"/abs/path":                          false,
	}
	for pkg, want := range tests {
		if got := IsLegacyPackage(pkg); got != want {
			t.Errorf("IsLegacyPackage(%q) = %v, want %v", pkg, got, want)
		}
	}
}

func TestSystemInfoOutputs(t *testing.T) {
	var nilInfo *SystemInfo
	if got := nilInfo.DefaultOutputs(); got != nil {
		t.Errorf("nil DefaultOutputs() = %v, want nil", got)
	}
	if _, err := nilInfo.Output("out"); err != nil {
		t.Errorf("nil Output() returned error %v", err)
	}
	if !nilInfo.Equals(nil) || nilInfo.Equals(&SystemInfo{}) {
		t.Error("nil SystemInfo Equals() is wrong")
	}

	info := &SystemInfo{Outputs: []Output{
		{Name: "out", Path: "/nix/store/out"},
		{Name: "bin", Path: "/nix/store/bin", Default: true},
		{Name: "dev", Path: "/nix/store/dev", Default: true},
	}}
	if got, want := info.DefaultOutputs(), info.Outputs[1:]; !slices.Equal(got, want) {
		t.Errorf("DefaultOutputs() = %v, want %v", got, want)
	}
	if out, err := info.Output("dev"); err != nil || out.Path != "/nix/store/dev" {
		t.Errorf("Output(dev) = %v, %v", out, err)
	}
	if _, err := info.Output("lib"); err == nil {
		t.Error("Output(lib) of a package without lib returned no error")
	}

	// With no default outputs, the first one is used.
	noDefault := &SystemInfo{Outputs: []Output{{Name: "lib"}, {Name: "out"}}}
	if got := noDefault.DefaultOutputs(); len(got) != 1 || got[0].Name != "lib" {
		t.Errorf("DefaultOutputs() with no defaults = %v, want [lib]", got)
	}

	var nilPkg *Package
	if nilPkg.GetSource() != "" || nilPkg.IsAllowInsecure() {
		t.Error("nil Package accessors should return zero values")
	}
}
