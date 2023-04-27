// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"golang.org/x/mod/semver"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/envir"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
)

// Keep this in-sync with latest version in launch.sh. If this version is newer
// Than the version in launch.sh, we'll print a warning.
const expectedLauncherVersion = "v0.1.0"

// currentChannel represents the CLI update channel. For now, we set it to a
// constant, but we'll expand this to be a variable in the future.
const currentChannel = "stable"

func CheckLauncherVersion(w io.Writer) {
	if envir.IsDevboxCloud() {
		return
	}

	if isNewLauncherAvailable() {
		ux.Fwarning(
			w,
			"newer launcher version %s is available (current = %s), please update "+
				"using `devbox version update`\n",
			expectedLauncherVersion,
			currentLauncherVersion(),
		)
	}
}

// SelfUpdate updates the devbox launcher and binary. It ignores and deletes the
// version cache
func SelfUpdate(stdOut, stdErr io.Writer) error {
	if isNewLauncherAvailable() {
		return selfUpdateLauncher(stdOut, stdErr)
	}

	return selfUpdateBinary(stdOut, stdErr)
}

func selfUpdateLauncher(stdOut, stdErr io.Writer) error {
	installScript := ""
	if _, err := exec.LookPath("curl"); err == nil {
		installScript = "curl -fsSL https://get.jetpack.io/devbox | bash"
	} else if _, err := exec.LookPath("wget"); err == nil {
		installScript = "wget -qO- https://get.jetpack.io/devbox | bash"
	} else {
		return usererr.New("curl or wget is required to update devbox. Please install either and try again.")
	}

	// Delete version cache.
	_ = os.Remove(versionCacheFilePath())

	cmd := exec.Command("sh", "-c", installScript)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	fmt.Fprint(stdOut, "Latest version: ")
	return triggerUpdateAndPrintNewVersion(stdOut, stdErr)
}

// selfUpdateBinary will update the binary to the latest version.
func selfUpdateBinary(stdOut, stdErr io.Writer) error {
	resp, err := http.Get(fmt.Sprintf("https://releases.jetpack.io/devbox/%s/version", currentChannel))
	if err != nil {
		fmt.Fprint(
			stdErr,
			"Failed to get latest version. "+
				"Please try again, or run the command from https://www.jetpack.io/devbox/docs/installing_devbox/",
		)
		return errors.WithStack(err)
	}
	defer resp.Body.Close()

	newVersion, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.WithStack(err)
	}

	// Remove the version cache file, so that when we re-create it the expiration
	// time is reset.
	_ = os.Remove(versionCacheFilePath())

	// Write the new version to the version cache file.
	if err := os.WriteFile(versionCacheFilePath(), newVersion, 0644); err != nil {
		return errors.WithStack(err)
	}

	return triggerUpdateAndPrintNewVersion(stdOut, stdErr)
}

func triggerUpdateAndPrintNewVersion(stdOut, stdErr io.Writer) error {
	fmt.Fprintln(stdOut, "triggering update...")

	exe, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(exe, "version")
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	return errors.WithStack(cmd.Run())

}

// isNewLauncherAvailable returns the latest launcher version if it is
// available, or empty string if it is not.
func isNewLauncherAvailable() bool {
	launcherVersion := currentLauncherVersion()
	if launcherVersion == "" {
		return false
	}

	// If launcherVersion is invalid, this will return 0, and we'll assume that
	// a new launcher is not available.
	if semver.Compare(launcherVersion, expectedLauncherVersion) >= 0 {
		return false
	}

	return true
}

func currentLauncherVersion() string {
	launcherVersion := os.Getenv(envir.LauncherVersion)
	if launcherVersion == "" {
		return ""
	}
	return "v" + launcherVersion
}

// versionCacheFilePath returns the path to the file that contains the latest
// version. The launcher checks this file to see if a new version is available.
// If the version is newer, then the launcher updates.
//
// Note: keep this in sync with launch.sh code
func versionCacheFilePath() string {
	return filepath.Join(xdg.CacheSubpath("devbox"), "latest-version")
}
