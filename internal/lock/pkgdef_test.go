// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"go.jetify.com/devbox/internal/boxcli/featureflag"
)

// writePkgDef writes tool.pkg.json, whose artifact URL is
// file://vendor/<artifact>-<version>.
func writePkgDef(t *testing.T, dir, artifact string) {
	t.Helper()
	content := fmt.Sprintf(`{
  "schema": 1, "name": "tool",
  "platforms": ["aarch64-darwin", "x86_64-darwin", "aarch64-linux", "x86_64-linux"],
  "versions": {"type": "static", "versions": ["1.0.0", "1.1.0"]},
  "install": {"type": "binary", "url": "file://vendor/%s-{{.Version}}", "checksum": {"type": "none"}}
}`, artifact)
	if err := os.WriteFile(filepath.Join(dir, "tool.pkg.json"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestResolvePkgDefLastModified(t *testing.T) {
	featureflag.PkgDefs.EnableForTest(t)
	dir := t.TempDir()
	writePkgDef(t, dir, "tool")
	f, err := GetFile(&lockTestProject{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	const raw = "pkg:./tool.pkg.json@1.0"

	first, err := f.resolvePkgDef(t.Context(), raw, false)
	if err != nil {
		t.Fatal(err)
	}
	if first.LastModified == "" {
		t.Fatal("last_modified is not set, so lockfile sync would skip the entry")
	}

	// Re-resolving to the same entry keeps last_modified.
	first.LastModified = "2020-01-01T00:00:00Z"
	f.Packages[raw] = first
	again, err := f.resolvePkgDef(t.Context(), raw, true)
	if err != nil {
		t.Fatal(err)
	}
	if again.LastModified != first.LastModified {
		t.Errorf("last_modified = %q, want unchanged %q", again.LastModified, first.LastModified)
	}
}

// TestRelockPkgDef checks that an edited local definition is re-locked at
// the locked version, with the new artifact URL.
func TestRelockPkgDef(t *testing.T) {
	featureflag.PkgDefs.EnableForTest(t)
	dir := t.TempDir()
	writePkgDef(t, dir, "tool")
	f, err := GetFile(&lockTestProject{dir: dir})
	if err != nil {
		t.Fatal(err)
	}
	const raw = "pkg:./tool.pkg.json@1"
	// Lock 1.0.0 although 1.1.0 is the newest match for "1".
	locked, err := f.resolvePkgDef(t.Context(), "pkg:./tool.pkg.json@1.0", false)
	if err != nil {
		t.Fatal(err)
	}
	f.Packages[raw] = locked

	writePkgDef(t, dir, "renamed")
	relocked, err := f.RelockPkgDef(t.Context(), raw)
	if err != nil {
		t.Fatal(err)
	}
	if relocked.Version != "1.0.0" {
		t.Errorf("version = %q, want the locked 1.0.0", relocked.Version)
	}
	if relocked.DefinitionHash == locked.DefinitionHash {
		t.Error("definition_hash did not change")
	}
	for system, info := range relocked.Systems {
		if want := "file://vendor/renamed-1.0.0"; info.URL != want {
			t.Errorf("%s url = %q, want %q", system, info.URL, want)
		}
	}
	if f.Packages[raw] != relocked {
		t.Error("the lockfile entry was not replaced")
	}
}
