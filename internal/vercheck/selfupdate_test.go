// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/launcher"
)

// `devbox version update` is how users get a release with a fix, so if a
// release breaks it, users are stuck on that release. These tests run
// SelfUpdate against a local release channel, with the new version already
// downloaded so that nothing is fetched from GitHub.

const selfUpdateLatest = "0.30.1"

type selfUpdateEnv struct {
	launcherPath string
	newBinary    string
}

func setupSelfUpdate(t *testing.T, channelBody string, channelStatus int) *selfUpdateEnv {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envir.XDGCacheHome, filepath.Join(home, "xdg-cache"))
	t.Setenv(envir.XDGConfigHome, filepath.Join(home, "xdg-config"))
	t.Setenv(envir.DevboxUseVersion, "")
	t.Setenv(envir.DevboxLatestVersion, "")

	oldVersion := currentDevboxVersion
	currentDevboxVersion = "0.30.0"
	t.Cleanup(func() { currentDevboxVersion = oldVersion })

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(channelStatus)
		_, _ = w.Write([]byte(channelBody))
	}))
	t.Cleanup(server.Close)
	// The channel file points the launcher at another release channel; here,
	// the local server.
	writeTestFile(t, filepath.Join(home, "xdg-config", "devbox", "channel"), server.URL+"/channel", 0o644)

	env := &selfUpdateEnv{
		launcherPath: filepath.Join(home, "usr-local-bin", "devbox"),
		newBinary:    launcher.BinaryPath(selfUpdateLatest),
	}
	writeTestFile(t, env.launcherPath, "old launcher", 0o755)
	writeTestFile(t, env.newBinary, "devbox "+selfUpdateLatest, 0o755)
	t.Setenv(envir.LauncherPath, env.launcherPath)
	t.Setenv(envir.LauncherVersion, launcher.Version)
	return env
}

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(data))
}

func TestSelfUpdateSwitchesToLatestVersion(t *testing.T) {
	env := setupSelfUpdate(t, selfUpdateLatest+"\n", http.StatusOK)

	var stderr bytes.Buffer
	if err := SelfUpdate(&stderr, false); err != nil {
		t.Fatalf("SelfUpdate() error: %v\n%s", err, stderr.String())
	}

	cache := os.Getenv(envir.XDGCacheHome)
	if got := readTestFile(t, filepath.Join(cache, "devbox", "current-version")); got != selfUpdateLatest {
		t.Errorf("current-version = %q, want %q", got, selfUpdateLatest)
	}
	if got := readTestFile(t, filepath.Join(cache, "devbox", "latest-version")); got != selfUpdateLatest {
		t.Errorf("latest-version = %q, want %q", got, selfUpdateLatest)
	}
	if !strings.Contains(stderr.String(), "updated to Devbox version "+selfUpdateLatest) {
		t.Errorf("stderr doesn't report the update:\n%s", stderr.String())
	}
	// The launcher is up to date, so it's left alone (replacing it can need
	// sudo).
	if got := readTestFile(t, env.launcherPath); got != "old launcher" {
		t.Errorf("launcher was replaced although it's up to date: %q", got)
	}
}

func TestSelfUpdateReplacesOutdatedLauncher(t *testing.T) {
	env := setupSelfUpdate(t, selfUpdateLatest, http.StatusOK)
	// A bash launcher from before devbox had a built-in one.
	t.Setenv(envir.LauncherVersion, "0.2.2")

	var stderr bytes.Buffer
	if err := SelfUpdate(&stderr, false); err != nil {
		t.Fatalf("SelfUpdate() error: %v\n%s", err, stderr.String())
	}
	if got := readTestFile(t, env.launcherPath); got != "devbox "+selfUpdateLatest {
		t.Errorf("launcher = %q, want it replaced by devbox %s", got, selfUpdateLatest)
	}
	fi, err := os.Stat(env.launcherPath)
	if err != nil {
		t.Fatal(err)
	}
	if fi.Mode().Perm()&0o111 == 0 {
		t.Errorf("replaced launcher isn't executable: %v", fi.Mode())
	}
	if !strings.Contains(stderr.String(), "replaced the launcher at "+env.launcherPath) {
		t.Errorf("stderr doesn't report replacing the launcher:\n%s", stderr.String())
	}
}

func TestSelfUpdateLauncherFlagReplacesUpToDateLauncher(t *testing.T) {
	env := setupSelfUpdate(t, selfUpdateLatest, http.StatusOK)

	var stderr bytes.Buffer
	if err := SelfUpdate(&stderr, true); err != nil {
		t.Fatalf("SelfUpdate(updateLauncher=true) error: %v\n%s", err, stderr.String())
	}
	if got := readTestFile(t, env.launcherPath); got != "devbox "+selfUpdateLatest {
		t.Errorf("launcher = %q, want it replaced by devbox %s", got, selfUpdateLatest)
	}
}

func TestSelfUpdateAlreadyLatest(t *testing.T) {
	setupSelfUpdate(t, selfUpdateLatest, http.StatusOK)
	currentDevboxVersion = selfUpdateLatest

	var stderr bytes.Buffer
	if err := SelfUpdate(&stderr, false); err != nil {
		t.Fatalf("SelfUpdate() error: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "already at Devbox version "+selfUpdateLatest) {
		t.Errorf("stderr doesn't say devbox is up to date:\n%s", stderr.String())
	}
}

func TestSelfUpdateWarnsAboutUseVersion(t *testing.T) {
	setupSelfUpdate(t, selfUpdateLatest, http.StatusOK)
	t.Setenv(envir.DevboxUseVersion, "0.29.0")

	var stderr bytes.Buffer
	if err := SelfUpdate(&stderr, false); err != nil {
		t.Fatalf("SelfUpdate() error: %v\n%s", err, stderr.String())
	}
	if !strings.Contains(stderr.String(), "DEVBOX_USE_VERSION=0.29.0 is set") {
		t.Errorf("stderr doesn't warn that DEVBOX_USE_VERSION pins the version:\n%s", stderr.String())
	}
}

func TestSelfUpdateChannelError(t *testing.T) {
	env := setupSelfUpdate(t, "not found", http.StatusNotFound)

	var stderr bytes.Buffer
	err := SelfUpdate(&stderr, true)
	if err == nil {
		t.Fatal("SelfUpdate() with an unreachable release channel returned no error")
	}
	if !strings.Contains(err.Error(), "Failed to get the latest devbox version") {
		t.Errorf("error = %q, want it to explain that the latest version is unknown", err)
	}
	// Nothing changes when the update fails.
	if got := readTestFile(t, env.launcherPath); got != "old launcher" {
		t.Errorf("launcher changed after a failed update: %q", got)
	}
	if _, err := os.Stat(filepath.Join(os.Getenv(envir.XDGCacheHome), "devbox", "current-version")); !os.IsNotExist(err) {
		t.Errorf("current-version was written after a failed update (stat err = %v)", err)
	}
}

func TestSelfUpdateWithoutLauncher(t *testing.T) {
	setupSelfUpdate(t, selfUpdateLatest, http.StatusOK)
	// Installed by a package manager (nix, brew, ...), not install.sh.
	t.Setenv(envir.LauncherPath, "")

	err := SelfUpdate(&bytes.Buffer{}, false)
	if err == nil {
		t.Fatal("SelfUpdate() without a launcher returned no error")
	}
	if !strings.Contains(err.Error(), "install.sh") {
		t.Errorf("error = %q, want it to point at install.sh", err)
	}
}
