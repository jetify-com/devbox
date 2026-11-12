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

func TestSopsEnvFrom(t *testing.T) {
	rootPath := filepath.Join("/project", "devbox.json")
	tests := []struct {
		envFrom  string
		sops     bool
		dotEnv   bool
		sopsPath string
	}{
		{envFrom: "sops:secrets.yaml", sops: true, sopsPath: "/project/secrets.yaml"},
		{envFrom: "sops:sub/secrets.env", sops: true, sopsPath: "/project/sub/secrets.env"},
		{envFrom: "sops:/etc/secrets.json", sops: true, sopsPath: "/etc/secrets.json"},
		{envFrom: "sops: secrets.json", sops: true, sopsPath: "/project/secrets.json"},
		{envFrom: "secrets.env", dotEnv: true},
		{envFrom: "jetify-cloud"},
	}
	for _, test := range tests {
		t.Run(test.envFrom, func(t *testing.T) {
			c := &ConfigFile{EnvFrom: test.envFrom, AbsRootPath: rootPath}
			if got := c.IsSopsEnabled(); got != test.sops {
				t.Errorf("IsSopsEnabled() = %v, want %v", got, test.sops)
			}
			if got := c.IsdotEnvEnabled(); got != test.dotEnv {
				t.Errorf("IsdotEnvEnabled() = %v, want %v", got, test.dotEnv)
			}
			if test.sops {
				if got := c.SopsFilePath(); got != test.sopsPath {
					t.Errorf("SopsFilePath() = %q, want %q", got, test.sopsPath)
				}
			}
		})
	}
}
