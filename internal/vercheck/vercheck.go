// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/mod/semver"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
)

// Keep this in-sync with latest version in launch.sh.
// If this version is newer than the version in launch.sh, we'll print a notice.
const expectedLauncherVersion = "v0.2.2"

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

// SelfUpdate updates the devbox launcher and devbox CLI binary.
// It ignores and deletes the version cache.
//
// The launcher is a wrapper bash script introduced to manage the auto-update process
// for devbox. The production devbox application is actually this launcher script
// that acts as "devbox" and delegates commands to the devbox CLI binary.
func SelfUpdate(stdOut, stdErr io.Writer) error {
	if isNewLauncherAvailable() {
		return selfUpdateLauncher(stdOut, stdErr)
	}

	return selfUpdateDevbox(stdErr)
}

func selfUpdateLauncher(stdOut, stdErr io.Writer) error {
	installScript := ""
	if cmdutil.Exists("curl") {
		installScript = "curl -fsSL https://get.jetpack.io/devbox | bash"
	} else if cmdutil.Exists("wget") {
		installScript = "wget -qO- https://get.jetpack.io/devbox | bash"
	} else {
		return usererr.New("curl or wget is required to update devbox. Please install either and try again.")
	}

	// Delete current version file. This will trigger an update when invoking any devbox command;
	// in this case, inside triggerUpdate function.
	if err := removeCurrentVersionFile(); err != nil {
		return err
	}

	// Fetch the new launcher. And installs the new devbox CLI binary.
	cmd := exec.Command("sh", "-c", installScript)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	// Previously, we have already updated the binary. So, we call triggerUpdate
	// just to get the new version information.
	updated, err := triggerUpdate(stdErr)
	if err != nil {
		return errors.WithStack(err)
	}

	printSuccessMessage(stdErr, "Launcher", currentLauncherVersion(), updated.launcherVersion)
	printSuccessMessage(stdErr, "Devbox", currentDevboxVersion, updated.devboxVersion)

	return nil
}

// selfUpdateDevbox will update the devbox CLI binary to the latest version.
func selfUpdateDevbox(stdErr io.Writer) error {
	// Delete current version file. This will trigger an update when the next devbox command is run;
	// in this case, inside triggerUpdate function.
	if err := removeCurrentVersionFile(); err != nil {
		return err
	}

	updated, err := triggerUpdate(stdErr)
	if err != nil {
		return errors.WithStack(err)
	}

	printSuccessMessage(stdErr, "Devbox", currentDevboxVersion, updated.devboxVersion)

	return nil
}

type updatedVersions struct {
	devboxVersion   string
	launcherVersion string
}

// triggerUpdate runs `devbox version -v` and triggers an update since a new
// version is available. It parses the output to get the new launcher and
// devbox versions.
func triggerUpdate(stdErr io.Writer) (*updatedVersions, error) {
	exePath := os.Getenv(envir.LauncherPath)
	if exePath == "" {
		ux.Fwarning(stdErr, "expected LAUNCHER_PATH to be set. Defaulting to \"devbox\".")
		exePath = "devbox"
	}

	// TODO savil. Add a --json flag to devbox version and parse the output as JSON
	cmd := exec.Command(exePath, "version", "-v")

	buf := new(bytes.Buffer)
	cmd.Stdout = io.MultiWriter(stdErr, buf)
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return nil, errors.WithStack(err)
	}

	// Parse the output to ascertain the new devbox and launcher versions
	updated := &updatedVersions{}
	for _, line := range strings.Split(buf.String(), "\n") {
		if strings.HasPrefix(line, "Version:") {
			updated.devboxVersion = strings.TrimSpace(strings.TrimPrefix(line, "Version:"))
		}

		if strings.HasPrefix(line, "Launcher:") {
			updated.launcherVersion = strings.TrimSpace(strings.TrimPrefix(line, "Launcher:"))
		}
	}
	return updated, nil
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

	return fmt.Sprintf(
		"New launcher available: %s -> %s. Please run `devbox version update`.\n",
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

// currentLauncherAvailable returns launcher's version if it is
// available, or empty string if it is not.
func currentLauncherVersion() string {
	launcherVersion := os.Getenv(envir.LauncherVersion)
	if launcherVersion == "" {
		return ""
	}
	return "v" + launcherVersion
}

func removeCurrentVersionFile() error {
	// currentVersionFilePath is the path to the file that contains the cached
	// version. The launcher checks this file to see if a new version is available.
	// If the version is newer, then the launcher updates.
	//
	// Note: keep this in sync with launch.sh code
	currentVersionFilePath := filepath.Join(xdg.CacheSubpath("devbox"), "current-version")

	if err := os.Remove(currentVersionFilePath); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return usererr.WithLoggedUserMessage(
			err,
			"Failed to delete version-cache at %s. Please manually delete it and try again.",
			currentVersionFilePath,
		)
	}
	return nil
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
