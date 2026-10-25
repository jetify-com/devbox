// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenDocsAbsolutePath(t *testing.T) {
	// Run from an unrelated directory so a mangled path can't land in the
	// source tree.
	t.Chdir(t.TempDir())
	docsDir := t.TempDir()

	root := RootCmd()
	root.SetArgs([]string{"gen-docs", docsDir})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(docsDir, "devbox.md")); err != nil {
		t.Errorf("expected docs in %s: %v", docsDir, err)
	}
}
