package vercheck

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
	"golang.org/x/mod/semver"
)

// Keep this in-sync with latest version in launch.sh. If this version is newer
// Than the version in launch.sh, we'll print a warning.
const expectedLauncherVersion = "v0.1.0"

func CheckLauncherVersion(w io.Writer) {
	launcherVersion := os.Getenv("LAUNCHER_VERSION")
	if launcherVersion == "" {
		return // Launcher version will not be set in dev.
	}

	// If launcherVersion is invalid, this will return 0 and we won't print a warning
	if semver.Compare("v"+launcherVersion, expectedLauncherVersion) < 0 {
		ux.Fwarning(
			w,
			"newer launcher version %s is available (current = v%s), please update "+
				"using `devbox version update`\n",
			expectedLauncherVersion,
			os.Getenv("LAUNCHER_VERSION"),
		)
	}
}

// SelfUpdate updates the devbox launcher and binary. It ignores and deletes the
// version cache
func SelfUpdate(stdOut, stdErr io.Writer) error {
	// Delete version cache. Keep this in-sync with whatever logic is in launch.sh
	cacheDir := xdg.CacheSubpath("devbox")
	versionCacheFile := filepath.Join(cacheDir, "latest-version")
	_ = os.Remove(versionCacheFile)

	installScript := "curl -fsSL https://get.jetpack.io/devbox | bash"
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
