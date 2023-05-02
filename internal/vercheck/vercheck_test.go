package vercheck

import (
	"bytes"
	"strings"
	"testing"
)

func TestCheckVersion(t *testing.T) {

	t.Run("no_devbox_cloud", func(t *testing.T) {
		// if devbox cloud
		t.Setenv("DEVBOX_REGION", "true")
		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
		t.Setenv("DEVBOX_REGION", "")
	})

	// no envir.LauncherVersion or latest-version env var
	t.Run("no_launcher_version_or_latest_version", func(t *testing.T) {
		t.Setenv("LAUNCHER_VERSION", "")
		t.Setenv(envDevboxLatestVersion, "")
		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})

	t.Run("launcher_version_outdated", func(t *testing.T) {
		// set older launcher version
		t.Setenv("LAUNCHER_VERSION", "v0.1.0")

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if !strings.Contains(buf.String(), "New launcher available") {
			t.Errorf("expected notice about new launcher version, got %q", buf.String())
		}
	})

	t.Run("binary_version_outdated", func(t *testing.T) {
		// set the launcher version so that it is not outdated
		t.Setenv("LAUNCHER_VERSION", strings.TrimPrefix(expectedLauncherVersion, "v"))

		// set the latest version to be greater the current binary version
		t.Setenv(envDevboxLatestVersion, "0.4.9")

		// mock the existing binary version
		currentDevboxVersion = "v0.4.8"

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if !strings.Contains(buf.String(), "New devbox available") {
			t.Errorf("expected notice about new devbox version, got %q", buf.String())
		}
	})

	t.Run("all_versions_up_to_date", func(t *testing.T) {

		// set the launcher version so that it is not outdated
		t.Setenv("LAUNCHER_VERSION", strings.TrimPrefix(expectedLauncherVersion, "v"))

		// mock the existing binary version
		currentDevboxVersion = "v0.4.8"

		// set the latest version to the same as the current binary version
		t.Setenv(envDevboxLatestVersion, "0.4.8")

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})
}
