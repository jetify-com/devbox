package configfile

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestParseEnvsFromDotEnv(t *testing.T) {
	dir := t.TempDir()
	rootPath := filepath.Join(dir, "devbox.json")

	t.Run("parses an existing .env file", func(t *testing.T) {
		envPath := filepath.Join(dir, "exists.env")
		if err := os.WriteFile(envPath, []byte("FOO=bar\nBAZ=qux\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		c := &ConfigFile{EnvFrom: "exists.env", AbsRootPath: rootPath}
		got, err := c.ParseEnvsFromDotEnv()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got["FOO"] != "bar" || got["BAZ"] != "qux" {
			t.Errorf("unexpected env map: %v", got)
		}
	})

	t.Run("missing file returns an os.ErrNotExist error", func(t *testing.T) {
		// Regression test for #2504: a missing env_from file must be
		// distinguishable from a genuine parse error so callers can warn and
		// continue instead of failing to enable the environment.
		c := &ConfigFile{EnvFrom: "does-not-exist.env", AbsRootPath: rootPath}
		_, err := c.ParseEnvsFromDotEnv()
		if err == nil {
			t.Fatal("expected an error for a missing file, got nil")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected error to wrap os.ErrNotExist, got: %v", err)
		}
	})
}
