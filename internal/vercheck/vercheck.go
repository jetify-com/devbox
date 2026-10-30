// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/mod/semver"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/launcher"
	"go.jetify.com/devbox/internal/ux"
	"go.jetify.com/devbox/internal/xdg"
)

// expectedLauncherVersion is the launcher version that this devbox expects.
// When the launcher that ran devbox is older, devbox prints a notice asking
// users to replace it with `devbox version update`.
const expectedLauncherVersion = "v" + launcher.Version

// launcherNoticeInterval limits how often devbox prints the new launcher
// notice. Replacing the launcher can need sudo, so it shouldn't nag on every
// command.
const launcherNoticeInterval = 24 * time.Hour

// envName determines whether the version check has already occurred.
// We set this env-var so that this devbox command invoking other devbox commands
// do not re-run the version check and print the notice again.
const envName = "__DEVBOX_VERSION_CHECK"

// currentDevboxVersion is the version of the devbox CLI binary that is currently running.
// We use this variable so that we can mock it in tests.
var currentDevboxVersion = build.Version

// isDevBuild determines whether this CLI binary was built during development, or published
// as a release.
// We use this variable so we can mock it in tests.
var isDevBuild = build.IsDev

var commandSkipList = []string{
	"devbox global shellenv",
	"devbox shellenv",
	"devbox version update",
	"devbox log",
}

// CheckVersion checks the launcher and binary versions and prints a notice if
// they are out of date.
//
// It will set the checkVersionEnvName to indicate that the version check was done.
// Callers should call ClearEnvVar after their work is done.
func CheckVersion(w io.Writer, commandPath string) {
	if isDevBuild {
		return
	}

	if os.Getenv(envName) == "1" {
		return
	}

	// Don't use up the once-a-day launcher notice on a --quiet command.
	if w == io.Discard {
		return
	}

	if envir.IsDevboxCloud() {
		return
	}

	hasSkipPrefix := lo.ContainsBy(
		commandSkipList,
		func(skipPath string) bool { return strings.HasPrefix(commandPath, skipPath) },
	)
	if hasSkipPrefix {
		return
	}

	// check launcher version
	launcherNotice := launcherVersionNotice()
	if launcherNotice != "" {
		ux.Finfo(w, launcherNotice)

		// fallthrough to alert the user about a new Devbox CLI binary being possibly available
	}

	// check devbox CLI version
	devboxNotice := devboxVersionNotice()
	if devboxNotice != "" {
		ux.Finfo(w, devboxNotice)
	}

	os.Setenv(envName, "1")
}

// SelfUpdate updates devbox to the latest version. It also replaces the
// launcher, the devbox that users run, when updateLauncher is true or when the
// launcher is older than this devbox expects. Replacing the launcher can need
// sudo, which is why it doesn't happen on every update.
func SelfUpdate(stdErr io.Writer, updateLauncher bool) error {
	launcherPath := os.Getenv(envir.LauncherPath)
	if launcherPath == "" {
		return usererr.New(
			"Devbox can't update itself because it wasn't installed with the devbox installer. " +
				"Update it with the package manager you installed it with, or reinstall it with " +
				"`curl -fsSL https://get.jetify.com/devbox | bash`.",
		)
	}

	ctx := context.Background()
	latest, err := launcher.FetchLatestVersion(ctx)
	if err != nil {
		return usererr.WithUserMessage(err, "Failed to get the latest devbox version.")
	}
	bin, err := launcher.Install(ctx, latest, stdErr)
	if err != nil {
		return usererr.WithUserMessage(err, "Failed to download devbox %s.", latest)
	}
	if err := launcher.SetLatestVersion(latest); err != nil {
		return errors.WithStack(err)
	}
	if err := launcher.SetCurrentVersion(latest); err != nil {
		return errors.WithStack(err)
	}
	printSuccessMessage(stdErr, "Devbox", currentDevboxVersion, latest)
	if useVersion := userChosenVersion(); useVersion != "" {
		ux.Fwarningf(stdErr, "%s=%s is set, so devbox will keep running version %s until you unset it.\n",
			envir.DevboxUseVersion, useVersion, useVersion)
	}

	if !updateLauncher && !isNewLauncherAvailable() {
		return nil
	}
	// The new launcher must be a devbox with a built-in launcher. This devbox
	// has one, and so does every later version.
	src, srcVersion := bin, latest
	if SemverCompare(latest, currentDevboxVersion) < 0 {
		if src, err = os.Executable(); err != nil {
			return errors.WithStack(err)
		}
		srcVersion = currentDevboxVersion
	}
	if err := launcher.ReplaceLauncher(src, launcherPath, stdErr); err != nil {
		return usererr.WithUserMessage(err, "Failed to replace the devbox launcher at %s.", launcherPath)
	}
	fmt.Fprintf(stdErr, "%sreplaced the launcher at %s with devbox %s\n",
		color.New(color.FgGreen).Sprint("Success: "), launcherPath, strings.TrimPrefix(srcVersion, "v"))
	return nil
}

func printSuccessMessage(w io.Writer, toolName, oldVersion, newVersion string) {
	var msg string
	if SemverCompare(oldVersion, newVersion) == 0 {
		msg = fmt.Sprintf("already at %s version %s", toolName, newVersion)
	} else {
		msg = fmt.Sprintf("updated to %s version %s", toolName, newVersion)
	}

	// Prints a <green>Success:</green> message to the writer.
	// Move to ux.Success. Not doing so to minimize merge-conflicts.
	fmt.Fprintf(w, "%s%s\n", color.New(color.FgGreen).Sprint("Success: "), msg)
}

func launcherVersionNotice() string {
	if !isNewLauncherAvailable() {
		return ""
	}

	// Print the notice at most once per interval.
	noticeFile := xdg.CacheSubpath("devbox/launcher-notice")
	if fi, err := os.Stat(noticeFile); err == nil && time.Since(fi.ModTime()) < launcherNoticeInterval {
		return ""
	}
	if err := os.MkdirAll(filepath.Dir(noticeFile), 0o755); err == nil {
		_ = os.WriteFile(noticeFile, nil, 0o644)
	}

	return fmt.Sprintf(
		"New launcher available: %s -> %s. Please run `devbox version update` (it may ask for sudo).\n",
		currentLauncherVersion(),
		expectedLauncherVersion,
	)
}

func devboxVersionNotice() string {
	if !isNewDevboxAvailable() {
		return ""
	}

	return fmt.Sprintf(
		"New devbox available: %s -> %s. Please run `devbox version update`.\n",
		currentDevboxVersion,
		latestVersion(),
	)
}

// isNewLauncherAvailable returns true if a new launcher version is available.
func isNewLauncherAvailable() bool {
	launcherVersion := currentLauncherVersion()
	if launcherVersion == "" {
		return false
	}
	return SemverCompare(launcherVersion, expectedLauncherVersion) < 0
}

// isNewDevboxAvailable returns true if a new devbox CLI binary version is available.
func isNewDevboxAvailable() bool {
	latest := latestVersion()
	if latest == "" {
		return false
	}
	return SemverCompare(currentDevboxVersion, latest) < 0
}

// currentLauncherVersion returns launcher's version if it is
// available, or empty string if it is not.
func currentLauncherVersion() string {
	launcherVersion := os.Getenv(envir.LauncherVersion)
	if launcherVersion == "" {
		return ""
	}
	return "v" + launcherVersion
}

func SemverCompare(ver1, ver2 string) int {
	if !strings.HasPrefix(ver1, "v") {
		ver1 = "v" + ver1
	}
	if !strings.HasPrefix(ver2, "v") {
		ver2 = "v" + ver2
	}
	return semver.Compare(ver1, ver2)
}

// latestVersion returns the latest version available for the binary.
func latestVersion() string {
	version := os.Getenv(envir.DevboxLatestVersion)
	if version == "" {
		return ""
	}
	return "v" + version
}
