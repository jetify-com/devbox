package configfile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
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
		c := &ConfigFile{AbsRootPath: rootPath}
		got, err := c.ParseEnvsFromDotEnv("exists.env")
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
		c := &ConfigFile{AbsRootPath: rootPath}
		_, err := c.ParseEnvsFromDotEnv("does-not-exist.env")
		if err == nil {
			t.Fatal("expected an error for a missing file, got nil")
		}
		if !errors.Is(err, os.ErrNotExist) {
			t.Errorf("expected error to wrap os.ErrNotExist, got: %v", err)
		}
	})
}

func TestEnvFromJSON(t *testing.T) {
	tests := map[string]EnvFrom{
		`".env"`:               {".env"},
		`""`:                   nil,
		`null`:                 nil,
		`[".env","exec:op x"]`: {".env", "exec:op x"},
		`[]`:                   {},
	}
	for data, want := range tests {
		t.Run(data, func(t *testing.T) {
			var got EnvFrom
			if err := json.Unmarshal([]byte(data), &got); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("Unmarshal(%s) = %q, want %q", data, got, want)
			}
		})
	}

	var got EnvFrom
	if err := json.Unmarshal([]byte(`1`), &got); err == nil {
		t.Error("expected an error for a number")
	}
	for value, want := range map[string]string{".env": `".env"`, ".env,exec:x": `[".env","exec:x"]`} {
		data, err := json.Marshal(EnvFrom(strings.Split(value, ",")))
		if err != nil || string(data) != want {
			t.Errorf("Marshal(%q) = %s, %v, want %s", value, data, err, want)
		}
	}
}

func TestEnvFromSources(t *testing.T) {
	tests := []struct {
		source  string
		command string
		exec    bool
		dotEnv  bool
	}{
		{source: "exec:sops decrypt s.yaml", command: "sops decrypt s.yaml", exec: true},
		{source: "exec: op inject ", command: "op inject", exec: true},
		{source: "exec:cat a.env", command: "cat a.env", exec: true},
		{source: "secrets.env", dotEnv: true},
		{source: "jetify-cloud"},
	}
	for _, test := range tests {
		t.Run(test.source, func(t *testing.T) {
			command, exec := ExecCommand(test.source)
			if command != test.command || exec != test.exec {
				t.Errorf("ExecCommand() = %q, %v, want %q, %v", command, exec, test.command, test.exec)
			}
			if got := IsDotEnvSource(test.source); got != test.dotEnv {
				t.Errorf("IsDotEnvSource() = %v, want %v", got, test.dotEnv)
			}
		})
	}
}
