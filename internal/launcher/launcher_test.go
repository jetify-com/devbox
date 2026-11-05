// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package launcher

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"go.jetify.com/devbox/internal/envir"
)

const (
	testVersion   = "0.18.4" // the version of the binary under test
	latestVersion = "0.19.0"
)

// fakeReleases serves the stable channel and release archives that hold a
// devbox binary whose contents name the version.
type fakeReleases struct {
	t        *testing.T
	latest   string
	versions map[string][]byte // archive by version
	badSum   bool
	requests []string
}

func (f *fakeReleases) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.requests = append(f.requests, r.URL.Path)
	if r.URL.Path == "/releases/stable/version" {
		if f.latest == "" {
			http.NotFound(w, r)
			return
		}
		fmt.Fprintln(w, f.latest)
		return
	}
	var version, file string
	if _, err := fmt.Sscanf(strings.ReplaceAll(strings.TrimPrefix(r.URL.Path, "/download/"), "/", " "), "%s %s", &version, &file); err != nil {
		http.NotFound(w, r)
		return
	}
	archive, ok := f.versions[version]
	if !ok {
		http.NotFound(w, r)
		return
	}
	archiveName := fmt.Sprintf("devbox_%s_%s.tar.gz", version, platform())
	switch file {
	case "checksums.txt":
		sum := sha256.Sum256(archive)
		if f.badSum {
			sum = sha256.Sum256([]byte("something else"))
		}
		fmt.Fprintf(w, "%s  devbox_%s_other_os.tar.gz\n", strings.Repeat("0", 64), version)
		fmt.Fprintf(w, "%s  %s\n", hex.EncodeToString(sum[:]), archiveName)
	case archiveName:
		_, _ = w.Write(archive)
	default:
		http.NotFound(w, r)
	}
}

func (f *fakeReleases) addVersion(version string) {
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gzw)
	content := []byte("devbox " + version)
	if err := tw.WriteHeader(&tar.Header{Name: "devbox", Mode: 0o755, Size: int64(len(content))}); err != nil {
		f.t.Fatal(err)
	}
	if _, err := tw.Write(content); err != nil {
		f.t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		f.t.Fatal(err)
	}
	if err := gzw.Close(); err != nil {
		f.t.Fatal(err)
	}
	f.versions[version] = buf.Bytes()
}

type execCall struct {
	argv0 string
	argv  []string
	env   map[string]string
}

type testEnv struct {
	home     string
	exe      string
	releases *fakeReleases
	execs    []execCall
	spawns   [][]string
}

// setup isolates the launcher's files, network, and process calls.
func setup(t *testing.T) *testEnv {
	t.Helper()
	env := &testEnv{home: t.TempDir()}
	t.Setenv("HOME", env.home)
	t.Setenv(envir.XDGCacheHome, filepath.Join(env.home, "xdg-cache"))
	t.Setenv(envir.XDGConfigHome, filepath.Join(env.home, "xdg-config"))
	for _, name := range []string{envir.DevboxUseVersion, envir.LauncherPath, envir.LauncherVersion, envir.DevboxLatestVersion} {
		t.Setenv(name, "")
	}

	env.exe = filepath.Join(env.home, "usr-local-bin", "devbox")
	writeExecutable(t, env.exe, "devbox "+testVersion)

	env.releases = &fakeReleases{t: t, latest: latestVersion, versions: map[string][]byte{}}
	env.releases.addVersion(latestVersion)
	server := httptest.NewServer(env.releases)
	t.Cleanup(server.Close)
	setVar(t, &releasesURL, server.URL+"/releases")
	setVar(t, &downloadURL, server.URL+"/download")

	setVar(t, &execFunc, func(argv0 string, argv, envv []string) error {
		call := execCall{argv0: argv0, argv: argv, env: map[string]string{}}
		for _, kv := range envv {
			name, value, _ := strings.Cut(kv, "=")
			call.env[name] = value
		}
		env.execs = append(env.execs, call)
		return nil
	})
	setVar(t, &spawnFunc, func(name string, args ...string) error {
		env.spawns = append(env.spawns, append([]string{name}, args...))
		return nil
	})
	return env
}

func setVar[T any](t *testing.T, v *T, value T) {
	t.Helper()
	old := *v
	t.Cleanup(func() { *v = old })
	*v = value
}

func writeExecutable(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeVersionFile(t *testing.T, path, version string, modTime time.Time) {
	t.Helper()
	if err := writeVersion(path, version); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(path, modTime, modTime); err != nil {
		t.Fatal(err)
	}
}

func (env *testEnv) run(t *testing.T) error {
	t.Helper()
	return run(env.exe, testVersion, []string{env.exe, "shell", "--pure"}, io.Discard)
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRunsItselfWhenItIsTheCurrentVersion(t *testing.T) {
	env := setup(t)
	writeVersionFile(t, currentVersionFile(), testVersion, time.Now())
	writeVersionFile(t, latestVersionFile(), latestVersion, time.Now())

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if len(env.execs) != 0 {
		t.Errorf("got exec %v, want none", env.execs)
	}
	if len(env.spawns) != 0 {
		t.Errorf("got version check %v, want none", env.spawns)
	}
	for name, want := range map[string]string{
		envir.LauncherPath:        env.exe,
		envir.LauncherVersion:     Version,
		envir.DevboxLatestVersion: latestVersion,
	} {
		if got := os.Getenv(name); got != want {
			t.Errorf("got %s=%q, want %q", name, got, want)
		}
	}
	if len(env.releases.requests) != 0 {
		t.Errorf("got requests %v, want none", env.releases.requests)
	}
}

func TestRunsInstalledCurrentVersion(t *testing.T) {
	env := setup(t)
	writeVersionFile(t, currentVersionFile(), "0.17.0", time.Now())
	bin := BinaryPath("0.17.0")
	writeExecutable(t, bin, "devbox 0.17.0")

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if len(env.execs) != 1 {
		t.Fatalf("got %d execs, want 1", len(env.execs))
	}
	call := env.execs[0]
	if call.argv0 != bin {
		t.Errorf("got exec of %s, want %s", call.argv0, bin)
	}
	if want := []string{bin, "shell", "--pure"}; !slices.Equal(call.argv, want) {
		t.Errorf("got argv %q, want %q", call.argv, want)
	}
	if call.env[envir.LauncherPath] != env.exe || call.env[envir.LauncherVersion] != Version {
		t.Errorf("got launcher env %q and %q, want %q and %q",
			call.env[envir.LauncherPath], call.env[envir.LauncherVersion], env.exe, Version)
	}
	if len(env.releases.requests) != 0 {
		t.Errorf("got requests %v, want none", env.releases.requests)
	}
}

func TestDownloadsMissingVersion(t *testing.T) {
	env := setup(t)
	env.releases.addVersion("0.17.0")
	writeVersionFile(t, currentVersionFile(), "0.17.0", time.Now())

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	bin := BinaryPath("0.17.0")
	if got := readFile(t, bin); got != "devbox 0.17.0" {
		t.Errorf("got installed binary %q, want %q", got, "devbox 0.17.0")
	}
	if len(env.execs) != 1 || env.execs[0].argv0 != bin {
		t.Errorf("got execs %v, want one of %s", env.execs, bin)
	}
	leftovers, _ := filepath.Glob(filepath.Join(installDir(), ".download-*"))
	if len(leftovers) != 0 {
		t.Errorf("got leftover download dirs %v", leftovers)
	}
}

func TestRejectsArchiveWithWrongChecksum(t *testing.T) {
	env := setup(t)
	env.releases.addVersion("0.17.0")
	env.releases.badSum = true
	writeVersionFile(t, currentVersionFile(), "0.17.0", time.Now())

	err := env.run(t)
	if err == nil || !strings.Contains(err.Error(), "checksum mismatch") {
		t.Fatalf("got error %v, want a checksum mismatch", err)
	}
	if InstalledBinary("0.17.0") != "" {
		t.Error("installed a binary with the wrong checksum")
	}
	if len(env.execs) != 0 {
		t.Errorf("got execs %v, want none", env.execs)
	}
}

func TestUseVersionOverridesCurrentVersion(t *testing.T) {
	env := setup(t)
	writeVersionFile(t, currentVersionFile(), testVersion, time.Now())
	bin := BinaryPath("0.17.0")
	writeExecutable(t, bin, "devbox 0.17.0")
	t.Setenv(envir.DevboxUseVersion, "v0.17.0")

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if len(env.execs) != 1 || env.execs[0].argv0 != bin {
		t.Errorf("got execs %v, want one of %s", env.execs, bin)
	}
}

func TestRejectsInvalidUseVersion(t *testing.T) {
	env := setup(t)
	t.Setenv(envir.DevboxUseVersion, "../../evil")

	if err := env.run(t); err == nil {
		t.Fatal("got no error for an invalid DEVBOX_USE_VERSION")
	}
	if len(env.execs) != 0 {
		t.Errorf("got execs %v, want none", env.execs)
	}
}

func TestDoesNothingForInstalledVersions(t *testing.T) {
	env := setup(t)
	// The bash launcher (or another devbox) sets DEVBOX_USE_VERSION and runs a
	// binary from the install dir. That binary must run itself.
	t.Setenv(envir.DevboxUseVersion, "0.17.0")
	bin := BinaryPath(testVersion)
	writeExecutable(t, bin, "devbox "+testVersion)

	if err := run(bin, testVersion, []string{bin, "version"}, io.Discard); err != nil {
		t.Fatal(err)
	}
	if len(env.execs) != 0 {
		t.Errorf("got execs %v, want none", env.execs)
	}
	if got := os.Getenv(envir.LauncherPath); got != "" {
		t.Errorf("got %s=%q, want it left alone", envir.LauncherPath, got)
	}
}

func TestFirstRunPinsLatestVersion(t *testing.T) {
	env := setup(t)

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if got := readVersion(currentVersionFile()); got != latestVersion {
		t.Errorf("got current version %q, want %q", got, latestVersion)
	}
	if got := readVersion(latestVersionFile()); got != latestVersion {
		t.Errorf("got latest version %q, want %q", got, latestVersion)
	}
	if len(env.execs) != 1 || env.execs[0].argv0 != BinaryPath(latestVersion) {
		t.Errorf("got execs %v, want one of %s", env.execs, BinaryPath(latestVersion))
	}
}

func TestFirstRunOfflineRunsItself(t *testing.T) {
	env := setup(t)
	env.releases.latest = ""

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if len(env.execs) != 0 {
		t.Errorf("got execs %v, want none", env.execs)
	}
	if _, err := os.Stat(currentVersionFile()); err == nil {
		t.Error("saved a current version, want the next run to try again")
	}
}

func TestFirstRunUsesChannel(t *testing.T) {
	env := setup(t)
	env.releases.addVersion("0.0.0-edge.2026-10-25")
	edge := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "0.0.0-edge.2026-10-25")
	}))
	t.Cleanup(edge.Close)
	if err := os.MkdirAll(filepath.Dir(channelFile()), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(channelFile(), []byte(edge.URL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if got := readVersion(currentVersionFile()); got != "0.0.0-edge.2026-10-25" {
		t.Errorf("got current version %q, want the edge version", got)
	}
}

func TestStaleVersionStartsBackgroundCheck(t *testing.T) {
	env := setup(t)
	writeVersionFile(t, currentVersionFile(), testVersion, time.Now().Add(-25*time.Hour))

	if err := env.run(t); err != nil {
		t.Fatal(err)
	}
	if want := [][]string{{env.exe, refreshArg}}; !slices.EqualFunc(env.spawns, want, slices.Equal) {
		t.Errorf("got spawns %v, want %v", env.spawns, want)
	}
	fi, err := os.Stat(currentVersionFile())
	if err != nil {
		t.Fatal(err)
	}
	if time.Since(fi.ModTime()) > time.Minute {
		t.Errorf("didn't reset the check interval: modified %v", fi.ModTime())
	}
	// The command must not wait for the network.
	if len(env.releases.requests) != 0 {
		t.Errorf("got requests %v, want none", env.releases.requests)
	}
	if len(env.execs) != 0 {
		t.Errorf("got execs %v, want none", env.execs)
	}
}

func TestRefresh(t *testing.T) {
	t.Run("saves latest version", func(t *testing.T) {
		setup(t)
		writeVersionFile(t, currentVersionFile(), testVersion, time.Now())

		refresh(context.Background())
		if got := readVersion(latestVersionFile()); got != latestVersion {
			t.Errorf("got latest version %q, want %q", got, latestVersion)
		}
		if got := readVersion(currentVersionFile()); got != testVersion {
			t.Errorf("got current version %q, want it unchanged at %q", got, testVersion)
		}
		if InstalledBinary(latestVersion) != "" {
			t.Error("downloaded the latest version without auto_update")
		}
	})

	t.Run("auto_update switches to latest version", func(t *testing.T) {
		setup(t)
		writeVersionFile(t, currentVersionFile(), testVersion, time.Now())
		if err := os.MkdirAll(filepath.Dir(autoUpdateFile()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(autoUpdateFile(), []byte("true\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		refresh(context.Background())
		if got := readVersion(currentVersionFile()); got != latestVersion {
			t.Errorf("got current version %q, want %q", got, latestVersion)
		}
		if InstalledBinary(latestVersion) == "" {
			t.Error("didn't download the latest version before switching to it")
		}
	})
}

func TestAutoUpdate(t *testing.T) {
	for content, want := range map[string]bool{"": false, "0": false, "FALSE\n": false, "1": true, "yes": true} {
		setup(t)
		if err := os.MkdirAll(filepath.Dir(autoUpdateFile()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(autoUpdateFile(), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := autoUpdate(); got != want {
			t.Errorf("auto_update %q: got %v, want %v", content, got, want)
		}
	}
}

func TestReplaceLauncher(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "new-devbox")
	writeExecutable(t, src, "devbox binary")
	real := filepath.Join(dir, "launch.sh")
	if err := os.WriteFile(real, []byte("#!/bin/bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(dir, "devbox")
	if err := os.Symlink(real, link); err != nil {
		t.Fatal(err)
	}

	if err := ReplaceLauncher(src, link, io.Discard); err != nil {
		t.Fatal(err)
	}
	if got := readFile(t, real); got != "devbox binary" {
		t.Errorf("got launcher %q, want the new binary", got)
	}
	if fi, err := os.Lstat(link); err != nil || fi.Mode()&os.ModeSymlink == 0 {
		t.Error("replaced the symlink instead of the file it points to")
	}
	if !isExecutable(real) {
		t.Error("new launcher isn't executable")
	}
}

func TestValidVersion(t *testing.T) {
	for _, v := range []string{"0.18.4", "v0.18.4", "0.0.0-edge.2026-10-25", "0.8.3-beta"} {
		if !validVersion(v) {
			t.Errorf("validVersion(%q) = false, want true", v)
		}
	}
	for _, v := range []string{"", "latest", "0.18", "0.18.4/../x", "0.18.4 ", "1.2.3-"} {
		if validVersion(v) {
			t.Errorf("validVersion(%q) = true, want false", v)
		}
	}
}
