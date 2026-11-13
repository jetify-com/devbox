// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWriteScriptFileAtomicSkipsUnchanged(t *testing.T) {
	path := filepath.Join(t.TempDir(), "script.sh")
	if err := writeScriptFileAtomic(path, "echo hi\n"); err != nil {
		t.Fatal(err)
	}

	// Backdate the file so that a rewrite is visible in its mtime.
	old := time.Now().Add(-time.Hour).Truncate(time.Second)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	if err := writeScriptFileAtomic(path, "echo hi\n"); err != nil {
		t.Fatal(err)
	}
	fi, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if !fi.ModTime().Equal(old) {
		t.Errorf("unchanged script was rewritten: mtime = %v, want %v", fi.ModTime(), old)
	}

	// Wrong permissions force a rewrite.
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeScriptFileAtomic(path, "echo hi\n"); err != nil {
		t.Fatal(err)
	}
	if fi, err := os.Stat(path); err != nil || fi.Mode().Perm() != 0o755 {
		t.Errorf("got mode %v (err %v), want 0755", fi.Mode().Perm(), err)
	}

	// Changed contents are written.
	if err := writeScriptFileAtomic(path, "echo bye\n"); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "echo bye\n" {
		t.Errorf("got %q, want %q", got, "echo bye\n")
	}
}
