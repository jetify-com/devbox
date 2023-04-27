// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"fmt"
	"io"
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

func CheckLauncherVersion(w io.Writer) {
	launcherVersion := os.Getenv(envir.LauncherVersion)
	if launcherVersion == "" || envir.IsDevboxCloud() {
		return
	}

	// If launcherVersion is invalid, this will return 0 and we won't print a warning
	if semver.Compare("v"+launcherVersion, expectedLauncherVersion) < 0 {
		ux.Fwarning(
			w,
			"newer launcher version %s is available (current = v%s), please update "+
				"using `devbox version update`\n",
			expectedLauncherVersion,
			launcherVersion,
		)
	}
}

// SelfUpdate updates the devbox launcher and binary. It ignores and deletes the
// version cache
func SelfUpdate(stdOut, stdErr io.Writer) error {
	installScript := ""
	if _, err := exec.LookPath("curl"); err == nil {
		installScript = "curl -fsSL https://get.jetpack.io/devbox | bash"
	} else if _, err := exec.LookPath("wget"); err == nil {
		installScript = "wget -qO- https://get.jetpack.io/devbox | bash"
	} else {
		return usererr.New("curl or wget is required to update devbox. Please install either and try again.")
	}

	// Delete version cache. Keep this in-sync with whatever logic is in launch.sh
	cacheDir := xdg.CacheSubpath("devbox")
	versionCacheFile := filepath.Join(cacheDir, "latest-version")
	_ = os.Remove(versionCacheFile)

	cmd := exec.Command("sh", "-c", installScript)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr
	if err := cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	fmt.Fprint(stdErr, "Latest version: ")
	exe, err := os.Executable()
	if err != nil {
		return errors.WithStack(err)
	}
	cmd = exec.Command(exe, "version")
	// The output of version is incidental, so just send it all to stdErr
	cmd.Stdout = stdErr
	cmd.Stderr = stdErr
	return errors.WithStack(cmd.Run())
}
