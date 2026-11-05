// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCreateDevboxSymlinkReplacesStaleLink(t *testing.T) {
	d := &Devbox{projectDir: t.TempDir()}
	binPath := dotdevboxBinPath(d)
	if err := os.MkdirAll(binPath, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(binPath, "devbox")
	if err := os.Symlink("/old/devbox", link); err != nil {
		t.Fatal(err)
	}

	if err := createDevboxSymlink(d); err != nil {
		t.Fatal("createDevboxSymlink error:", err)
	}

	want, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	got, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("devbox symlink points to %q, want %q", got, want)
	}
}
