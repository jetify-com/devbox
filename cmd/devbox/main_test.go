// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestReleaseBuildReportsVersion builds devbox the way .goreleaser.yaml does
// (same -X ldflags, with placeholder values) and checks what the binary
// reports. Release tests otherwise run a plain `go build`, so a mistake in the
// release ldflags or in how devbox reads them would ship a binary that reports
// 0.0.0-dev, or acts like a dev build, without failing CI.
func TestReleaseBuildReportsVersion(t *testing.T) {
	const (
		version    = "9.8.7-releasetest.1"
		commit     = "0123456789abcdef0123456789abcdef01234567"
		commitDate = "2026-01-02T03:04:05Z"
	)
	bin := buildRelease(t, map[string]string{
		".Version":           version,
		".Commit":            commit,
		".CommitDate":        commitDate,
		".Env.SENTRY_DSN":    "",
		".Env.TELEMETRY_KEY": "",
	})
	env := releaseEnv(t, version)

	stdout := runBin(t, bin, env, "version")
	if got := strings.TrimSpace(stdout); got != version {
		t.Errorf("`devbox version` = %q, want %q", got, version)
	}

	verbose := runBin(t, bin, env, "version", "--verbose")
	for _, want := range []string{
		"Version:     " + version,
		"Commit:      " + commit,
		"Commit Time: " + commitDate,
		"Platform:    " + runtime.GOOS + "_" + runtime.GOARCH,
	} {
		if !strings.Contains(verbose, want) {
			t.Errorf("`devbox version --verbose` output is missing %q:\n%s", want, verbose)
		}
	}
	// A release build launches itself, so it reports a launcher version.
	if !regexp.MustCompile(`(?m)^Launcher:\s+\d+\.\d+\.\d+$`).MatchString(verbose) {
		t.Errorf("`devbox version --verbose` doesn't report a launcher version; "+
			"is -X ...build.Launcher=true still in .goreleaser.yaml?\n%s", verbose)
	}
}

// TestReleaseBuildWithTelemetryKeys runs a release build that has telemetry
// and error reporting keys, like official releases do. Only release builds
// have them, so the code that records events and errors never runs in the rest
// of CI. The keys point at an unreachable local address, so nothing is sent.
func TestReleaseBuildWithTelemetryKeys(t *testing.T) {
	const version = "9.8.7-releasetest.2"
	bin := buildRelease(t, map[string]string{
		".Version":           version,
		".Commit":            "0123456789abcdef0123456789abcdef01234567",
		".CommitDate":        "2026-01-02T03:04:05Z",
		".Env.SENTRY_DSN":    "http://publickey@127.0.0.1:9/1",
		".Env.TELEMETRY_KEY": "releasetest-telemetry-key",
	})
	env := releaseEnv(t, version)
	env = slices.DeleteFunc(env, func(kv string) bool { return strings.HasPrefix(kv, "DO_NOT_TRACK=") })

	if got := strings.TrimSpace(runBin(t, bin, env, "version")); got != version {
		t.Errorf("`devbox version` = %q, want %q", got, version)
	}

	// A command that fails reports the error to Sentry. It must still fail
	// with the usual error and exit code, not crash or hang.
	var stderr bytes.Buffer
	cmd := exec.Command(bin, "run", "--list")
	cmd.Env = env
	cmd.Dir = t.TempDir() // no devbox.json
	cmd.Stderr = &stderr
	err := cmd.Run()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Errorf("`devbox run --list` without a devbox.json: got %v, want exit code 1\nstderr:\n%s", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "panic:") {
		t.Errorf("`devbox run --list` panicked:\n%s", stderr.String())
	}
	if !strings.Contains(stderr.String(), "Error:") {
		t.Errorf("`devbox run --list` didn't print an error:\n%s", stderr.String())
	}

	// Telemetry only starts when the build has both keys, and it then records
	// a device ID. Check it did, so that this test can't pass without
	// exercising the telemetry code.
	stateHome := strings.TrimPrefix(env[slices.IndexFunc(env, func(kv string) bool {
		return strings.HasPrefix(kv, "XDG_STATE_HOME=")
	})], "XDG_STATE_HOME=")
	if _, err := os.Stat(filepath.Join(stateHome, "devbox", "telemetry", "device-id")); err != nil {
		t.Errorf("telemetry didn't start in a build with telemetry keys: %v", err)
	}
}

// buildRelease builds devbox with the ldflags from .goreleaser.yaml, with its
// template fields replaced by vars.
func buildRelease(t *testing.T, vars map[string]string) string {
	t.Helper()
	if testing.Short() {
		t.Skip("builds the devbox binary")
	}
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not found in PATH")
	}
	ldflags := goreleaserLdflags(t, vars)
	bin := filepath.Join(t.TempDir(), "devbox")
	build := exec.Command(goBin, "build", "-ldflags", ldflags, "-o", bin, ".")
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build -ldflags %q failed: %v\n%s", ldflags, err, out)
	}
	return bin
}

// releaseEnv returns an environment for running a release build in isolation.
// Release builds are their own launcher; an isolated HOME and
// DEVBOX_USE_VERSION pinned to the binary's version make it run itself
// instead of looking for (and downloading) the latest release.
func releaseEnv(t *testing.T, version string) []string {
	t.Helper()
	home := t.TempDir()
	return []string{
		"HOME=" + home,
		"XDG_CACHE_HOME=" + filepath.Join(home, ".cache"),
		"XDG_CONFIG_HOME=" + filepath.Join(home, ".config"),
		"XDG_DATA_HOME=" + filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME=" + filepath.Join(home, ".local", "state"),
		"PATH=" + os.Getenv("PATH"),
		"DEVBOX_USE_VERSION=" + version,
		"DO_NOT_TRACK=1",
	}
}

func runBin(t *testing.T, bin string, env []string, args ...string) string {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cmd := exec.Command(bin, args...)
	cmd.Env = env
	cmd.Dir = t.TempDir()
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("devbox %s: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
	}
	return stdout.String()
}

// goreleaserLdflags returns the ldflags of the first build in .goreleaser.yaml
// with its {{ ... }} template fields replaced by vars.
func goreleaserLdflags(t *testing.T, vars map[string]string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "..", ".goreleaser.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var cfg struct {
		Builds []struct {
			Ldflags []string `yaml:"ldflags"`
		} `yaml:"builds"`
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if len(cfg.Builds) == 0 || len(cfg.Builds[0].Ldflags) == 0 {
		t.Fatal(".goreleaser.yaml has no build ldflags")
	}
	field := regexp.MustCompile(`\{\{\s*([^}\s]+)\s*\}\}`)
	flags := strings.Join(cfg.Builds[0].Ldflags, " ")
	flags = field.ReplaceAllStringFunc(flags, func(m string) string {
		name := field.FindStringSubmatch(m)[1]
		v, ok := vars[name]
		if !ok {
			t.Fatalf(".goreleaser.yaml ldflags use template field %s, which this test doesn't know about", name)
		}
		return v
	})
	return flags
}
