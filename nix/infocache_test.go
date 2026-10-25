package nix

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeNixVersion writes a fake nix executable that prints "nix --version --debug"
// output referencing confPath and records each invocation in a log file. It
// returns the path to the executable and a function that returns how many
// times it has run.
func fakeNixVersion(t *testing.T, confPath string, exitCode int) (path string, runs func() int) {
	t.Helper()

	dir := t.TempDir()
	logPath := filepath.Join(dir, "runs.log")
	path = filepath.Join(dir, "nix")
	script := `#!/bin/sh
echo run >> '` + logPath + `'
cat <<EOF
nix (Nix) 2.21.2
System type: aarch64-darwin
System configuration file: ` + confPath + `
User configuration files: ` + filepath.Join(dir, "missing.conf") + `
Store directory: /nix/store
EOF
exit ` + strconv.Itoa(exitCode) + `
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, func() int {
		data, _ := os.ReadFile(logPath)
		return strings.Count(string(data), "run")
	}
}

func setupInfoCacheTest(t *testing.T) (confPath string) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	for _, env := range infoCacheEnv {
		if env != "HOME" {
			t.Setenv(env, "")
		}
	}

	confPath = filepath.Join(t.TempDir(), "nix.conf")
	if err := os.WriteFile(confPath, []byte("experimental-features = nix-command\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return confPath
}

func TestInfoDiskCache(t *testing.T) {
	confPath := setupInfoCacheTest(t)
	nixPath, runs := fakeNixVersion(t, confPath, 0)

	info := func() Info {
		t.Helper()
		// Use a new Nix each time to bypass the in-process cache.
		info, err := (&Nix{Path: nixPath}).Info()
		if err != nil {
			t.Fatal("got Info error:", err)
		}
		return info
	}

	got := info()
	if got.Version != "2.21.2" || got.System != "aarch64-darwin" {
		t.Errorf("got Info = %+v, want version 2.21.2 and system aarch64-darwin", got)
	}
	if runs() != 1 {
		t.Fatalf("got %d nix runs, want 1", runs())
	}

	got = info()
	if got.Version != "2.21.2" || got.System != "aarch64-darwin" {
		t.Errorf("got cached Info = %+v, want version 2.21.2 and system aarch64-darwin", got)
	}
	if runs() != 1 {
		t.Errorf("got %d nix runs after a cache hit, want 1", runs())
	}

	// Editing nix.conf invalidates the cache.
	if err := os.WriteFile(confPath, []byte("system = x86_64-darwin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	future := time.Now().Add(time.Minute)
	if err := os.Chtimes(confPath, future, future); err != nil {
		t.Fatal(err)
	}
	info()
	if runs() != 2 {
		t.Errorf("got %d nix runs after editing nix.conf, want 2", runs())
	}

	// Creating a user config file that didn't exist invalidates the cache.
	if err := os.WriteFile(filepath.Join(filepath.Dir(nixPath), "missing.conf"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	info()
	if runs() != 3 {
		t.Errorf("got %d nix runs after creating a user nix.conf, want 3", runs())
	}

	// Changing an environment variable that affects nix's config changes
	// the cache key.
	t.Setenv("NIX_CONFIG", "system = x86_64-darwin")
	info()
	if runs() != 4 {
		t.Errorf("got %d nix runs after setting NIX_CONFIG, want 4", runs())
	}
	info()
	if runs() != 4 {
		t.Errorf("got %d nix runs after a cache hit, want 4", runs())
	}
}

func TestInfoDiskCacheSkipsErrors(t *testing.T) {
	confPath := setupInfoCacheTest(t)
	nixPath, runs := fakeNixVersion(t, confPath, 1)

	for range 2 {
		if _, err := (&Nix{Path: nixPath}).Info(); err == nil {
			t.Fatal("got nil Info error for a failing nix command")
		}
	}
	if runs() != 2 {
		t.Errorf("got %d nix runs, want 2 (failures shouldn't be cached)", runs())
	}
}
