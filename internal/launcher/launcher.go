// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

// Package launcher makes release builds of devbox their own launcher. The
// devbox that users run, such as /usr/local/bin/devbox, picks the devbox
// version to use, downloads it if needed, and runs it.
//
// It replaces the bash launcher that the installer used to put in
// /usr/local/bin. Older devbox versions rely on how that script behaves, so
// this package keeps its environment variables, version files, and download
// directory. See specs/native-launcher.md.
package launcher

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/envir"
)

// Version is the launcher version, which devbox passes to the version it runs
// as LAUNCHER_VERSION. Bump it only when installed launchers need to change,
// because a newer version makes devbox ask users to replace their launcher,
// which can need sudo. The bash launcher's versions are all below 1.0.0.
const Version = "1.0.0"

// refreshArg is a hidden command that checks for a new devbox version. The
// launcher runs it in a detached child process so that commands don't wait
// for the network.
const refreshArg = "__launcher-refresh"

// versionCheckInterval is how often the launcher checks for a new devbox
// version. It matches the bash launcher's default VERSION_CACHE_TTL.
const versionCheckInterval = 24 * time.Hour

const (
	// firstRunTimeout limits how long the first run waits to learn the latest
	// version. On a timeout, it runs the installed version.
	firstRunTimeout = 5 * time.Second
	// refreshTimeout limits how long the background version check runs,
	// including downloading the new version when auto_update is on.
	refreshTimeout = 5 * time.Minute
)

// These variables let tests replace process-level behavior.
var (
	execFunc  = syscall.Exec
	spawnFunc = spawnDetached
)

// Main runs the launcher in release builds. It returns when this process
// should run the command itself. Otherwise it replaces this process with the
// devbox version to use, and exits if it can't.
func Main() {
	if build.Launcher != "true" {
		return
	}
	if len(os.Args) > 1 && os.Args[1] == refreshArg {
		ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
		refresh(ctx)
		cancel()
		os.Exit(0)
	}
	exe, err := os.Executable()
	if err != nil {
		slog.Debug("launcher: can't find this binary, so running it as is", "err", err)
		return
	}
	if err := run(exe, build.Version, os.Args, os.Stderr); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run starts the devbox version to use for args, or returns nil if it's
// version, the version of the binary at exe.
func run(exe, version string, args []string, stderr io.Writer) error {
	// A launcher already picked this version: launchers only run binaries
	// from the install dir, and this binary is one of them.
	if inInstallDir(exe) {
		return nil
	}

	target, err := selectVersion(exe, version)
	if err != nil {
		return err
	}
	env := map[string]string{
		envir.LauncherVersion:     Version,
		envir.LauncherPath:        exe,
		envir.DevboxLatestVersion: readVersion(latestVersionFile()),
	}
	if sameVersion(target, version) {
		for name, value := range env {
			if err := os.Setenv(name, value); err != nil {
				return err
			}
		}
		return nil
	}

	bin, err := Install(context.Background(), target, stderr)
	if err != nil {
		return err
	}
	argv := append([]string{bin}, args[1:]...)
	if err := execFunc(bin, argv, withEnv(os.Environ(), env)); err != nil {
		return fmt.Errorf("run devbox %s from %s: %w", target, bin, err)
	}
	return nil
}

// selectVersion returns the devbox version to run: DEVBOX_USE_VERSION if it's
// set, or the version in the current-version file. It starts a background
// check for a new version when the last check was too long ago.
func selectVersion(exe, version string) (string, error) {
	if useVersion := os.Getenv(envir.DevboxUseVersion); useVersion != "" {
		if !validVersion(useVersion) {
			return "", fmt.Errorf("invalid %s %q", envir.DevboxUseVersion, useVersion)
		}
		return normalizeVersion(useVersion), nil
	}

	currentFile := currentVersionFile()
	if fi, err := os.Stat(currentFile); err == nil {
		if current := readVersion(currentFile); current != "" {
			if time.Since(fi.ModTime()) >= versionCheckInterval {
				startRefresh(exe, currentFile)
			}
			return current, nil
		}
	}

	// There's no current version. Either this is the first run after an
	// install that didn't write one, or an older devbox's `devbox version
	// update` deleted it to make the launcher pick the latest version. Like the
	// bash launcher, wait for the latest version and keep using it.
	ctx, cancel := context.WithTimeout(context.Background(), firstRunTimeout)
	defer cancel()
	latest, err := FetchLatestVersion(ctx)
	if err != nil {
		// Run this version without saving it, so the next run tries again.
		slog.Debug("launcher: can't get the latest devbox version", "err", err)
		return version, nil
	}
	if err := SetLatestVersion(latest); err != nil {
		slog.Debug("launcher: can't save the latest devbox version", "err", err)
	}
	if err := SetCurrentVersion(latest); err != nil {
		slog.Debug("launcher: can't save the current devbox version", "err", err)
	}
	return latest, nil
}

// startRefresh checks for a new devbox version in a detached child process.
func startRefresh(exe, currentFile string) {
	// Reset the check interval first, so that commands that start before the
	// check finishes don't start another one.
	now := time.Now()
	if err := os.Chtimes(currentFile, now, now); err != nil {
		slog.Debug("launcher: can't reset the version check interval", "err", err)
		return
	}
	if err := spawnFunc(exe, refreshArg); err != nil {
		slog.Debug("launcher: can't start the version check", "err", err)
	}
}

func spawnDetached(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// refresh saves the latest devbox version. When auto_update is on, it also
// downloads that version and makes it the current version.
func refresh(ctx context.Context) {
	latest, err := FetchLatestVersion(ctx)
	if err != nil {
		return
	}
	if err := SetLatestVersion(latest); err != nil || !autoUpdate() {
		return
	}
	// Download before switching, so that the next command doesn't wait for it.
	if _, err := Install(ctx, latest, io.Discard); err != nil {
		return
	}
	_ = SetCurrentVersion(latest)
}

// inInstallDir reports whether exe is a binary in the install dir.
func inInstallDir(exe string) bool {
	dir := installDir()
	if dir == "" {
		return false
	}
	if resolved, err := filepath.EvalSymlinks(dir); err == nil {
		dir = resolved
	}
	if resolved, err := filepath.EvalSymlinks(exe); err == nil {
		exe = resolved
	}
	rel, err := filepath.Rel(dir, exe)
	return err == nil && filepath.IsLocal(rel)
}

// withEnv returns environ with the variables in vars set, replacing any
// existing values.
func withEnv(environ []string, vars map[string]string) []string {
	result := make([]string, 0, len(environ)+len(vars))
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if _, ok := vars[name]; !ok {
			result = append(result, kv)
		}
	}
	for name, value := range vars {
		result = append(result, name+"="+value)
	}
	return result
}
