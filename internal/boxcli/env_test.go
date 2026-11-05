// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnvFileRelativeToConfig(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("FOO=bar\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	for _, configPath := range []string{dir, filepath.Join(dir, "devbox.json")} {
		f := &envFlag{EnvFile: ".env"}
		env, err := f.Env(configPath)
		if err != nil {
			t.Errorf("Env(%q) error: %v", configPath, err)
			continue
		}
		if env["FOO"] != "bar" {
			t.Errorf("Env(%q)[FOO] = %q, want %q", configPath, env["FOO"], "bar")
		}
	}
}
