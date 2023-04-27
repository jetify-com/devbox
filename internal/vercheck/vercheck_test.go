package vercheck

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.jetpack.io/devbox/internal/envir"
)

func TestCheckVersion(t *testing.T) {

	t.Run("no_devbox_cloud", func(t *testing.T) {
		// if devbox cloud
		t.Setenv(envir.DevboxRegion, "true")
		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
		t.Setenv(envir.DevboxRegion, "")
	})

	// no envir.LauncherVersion or available-version file
	t.Run("no_launcher_version_or_available_version_file", func(t *testing.T) {
		t.Setenv(envir.LauncherVersion, "")
		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})

	t.Run("launcher_version_outdated", func(t *testing.T) {
		// set older launcher version
		t.Setenv(envir.LauncherVersion, "v0.1.0")

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if !strings.Contains(buf.String(), "New launcher available") {
			t.Errorf("expected notice about new launcher version, got %q", buf.String())
		}
	})

	t.Run("binary_version_outdated", func(t *testing.T) {
		// set the launcher version so that it is not outdated
		t.Setenv(envir.LauncherVersion, strings.TrimPrefix(expectedLauncherVersion, "v"))

		// create the new available-version file
		setTestAvailableVersionFile(t, "v0.4.9")

		// mock the existing binary version
		currentBinaryVersion = "v0.4.8"

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if !strings.Contains(buf.String(), "New devbox available") {
			t.Errorf("expected notice about new devbox version, got %q", buf.String())
		}
	})

	t.Run("all_versions_up_to_date", func(t *testing.T) {

		// set the launcher version so that it is not outdated
		t.Setenv(envir.LauncherVersion, strings.TrimPrefix(expectedLauncherVersion, "v"))

		// mock the existing binary version
		currentBinaryVersion = "v0.4.8"

		// create the new available-version file
		setTestAvailableVersionFile(t, currentBinaryVersion)

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})
}

func setTestAvailableVersionFile(t *testing.T, version string) {

	xdgCacheDir := t.TempDir()
	t.Setenv(envir.XDGCacheHome, xdgCacheDir)
	path := availableVersionPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create directory: %v", err)
	}
	if err := os.WriteFile(path, []byte(version), 0644); err != nil {
		t.Fatalf("failed to write available-version file: %v", err)
	}
}
