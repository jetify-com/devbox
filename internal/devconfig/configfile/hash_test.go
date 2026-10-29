package configfile

import (
	"testing"
)

const hashTestBase = `{
  "$schema": "https://example.com/schema.json",
  "name": "my-project",
  "description": "a project",
  "devbox_version": ">=0.14.0",
  "packages": ["go@1.22", "hello@latest"],
  "env": {"FOO": "bar"},
  "env_from": ".env",
  "shell": {
    "init_hook": ["echo hello"],
    "scripts": {"test": "go test ./..."}
  },
  "aliases": {"ll": "ls -l"},
  "include": ["plugin:nginx"],
  "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}
}`

func mustLoad(t *testing.T, s string) *ConfigFile {
	t.Helper()
	cfg, err := LoadBytes([]byte(s))
	if err != nil {
		t.Fatalf("LoadBytes: %v", err)
	}
	return cfg
}

func mustHashes(t *testing.T, cfg *ConfigFile) (hash, nixHash string) {
	t.Helper()
	hash, err := cfg.Hash()
	if err != nil {
		t.Fatalf("Hash: %v", err)
	}
	nixHash, err = cfg.NixHash()
	if err != nil {
		t.Fatalf("NixHash: %v", err)
	}
	return hash, nixHash
}

func TestNixHash(t *testing.T) {
	baseHash, baseNixHash := mustHashes(t, mustLoad(t, hashTestBase))

	tests := []struct {
		name       string
		config     string
		nixChanged bool
	}{
		{
			name: "env",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "baz", "NEW": "1"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}}, "aliases": {"ll": "ls -l"},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}}`,
		},
		{
			name: "init_hook, scripts and env_from",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "bar"}, "env_from": "other.env",
				"shell": {"init_hook": ["echo bye"], "scripts": {"test": "go test -v ./...", "new": "true"}}, "aliases": {"ll": "ls -l"},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}}`,
		},
		{
			name: "metadata and aliases removed",
			config: `{"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "bar"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}}`,
		},
		{
			name: "packages",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.23", "hello@latest"], "env": {"FOO": "bar"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}}, "aliases": {"ll": "ls -l"},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}}`,
			nixChanged: true,
		},
		{
			name: "include",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "bar"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}}, "aliases": {"ll": "ls -l"},
				"include": [], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"}}`,
			nixChanged: true,
		},
		{
			name: "nixpkgs commit",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "bar"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}}, "aliases": {"ll": "ls -l"},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "1111111111111111111111111111111111111111"}}`,
			nixChanged: true,
		},
		{
			name: "unknown field is a nix input",
			config: `{"$schema": "https://example.com/schema.json", "name": "my-project", "description": "a project", "devbox_version": ">=0.14.0",
				"packages": ["go@1.22", "hello@latest"], "env": {"FOO": "bar"}, "env_from": ".env",
				"shell": {"init_hook": ["echo hello"], "scripts": {"test": "go test ./..."}}, "aliases": {"ll": "ls -l"},
				"include": ["plugin:nginx"], "nixpkgs": {"commit": "0000000000000000000000000000000000000000"},
				"some_future_field": true}`,
			nixChanged: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, nixHash := mustHashes(t, mustLoad(t, tt.config))
			if hash == baseHash {
				t.Errorf("Hash didn't change")
			}
			if gotChanged := nixHash != baseNixHash; gotChanged != tt.nixChanged {
				t.Errorf("NixHash changed = %v, want %v", gotChanged, tt.nixChanged)
			}
		})
	}
}

func TestNixHashIgnoresFormatting(t *testing.T) {
	_, want := mustHashes(t, mustLoad(t, hashTestBase))
	_, got := mustHashes(t, mustLoad(t, `{
  // A comment.
  "packages": [
    "go@1.22",
    "hello@latest",
  ],
  "include": ["plugin:nginx"],
  "nixpkgs": {"commit": "0000000000000000000000000000000000000000"},
  "env": {"FOO": "something else"}
}`))
	if got != want {
		t.Errorf("NixHash changed after reformatting and removing non-nix fields")
	}
}

func TestNixHashDoesNotModifyConfig(t *testing.T) {
	cfg := mustLoad(t, hashTestBase)
	before := string(cfg.Bytes())
	if _, err := cfg.NixHash(); err != nil {
		t.Fatal(err)
	}
	if after := string(cfg.Bytes()); after != before {
		t.Errorf("NixHash modified the config:\n%s", after)
	}
}

func TestNixHashWithoutAST(t *testing.T) {
	cfg := &ConfigFile{Env: map[string]string{"FOO": "bar"}}
	_, nixHash := mustHashes(t, cfg)

	cfg.Env["FOO"] = "baz"
	cfg.Shell = &shellConfig{}
	if _, got := mustHashes(t, cfg); got != nixHash {
		t.Errorf("NixHash changed after editing env and shell")
	}
	cfg.Nixpkgs = &NixpkgsConfig{Commit: "1111111111111111111111111111111111111111"}
	if _, got := mustHashes(t, cfg); got == nixHash {
		t.Errorf("NixHash didn't change after editing nixpkgs")
	}
}
