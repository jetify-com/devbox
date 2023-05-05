// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"bytes"
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

	// no launcher version or latest-version env var
	t.Run("no_launcher_version_or_latest_version", func(t *testing.T) {
		t.Setenv(envir.LauncherVersion, "")
		t.Setenv(envir.DevboxLatestVersion, "")
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

		// set the latest version to be greater the current binary version
		t.Setenv(envir.DevboxLatestVersion, "0.4.9")

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
		t.Setenv(envir.LauncherVersion, strings.TrimPrefix(expectedLauncherVersion, "v"))

		// mock the existing binary version
		currentDevboxVersion = "v0.4.8"

		// set the latest version to the same as the current binary version
		t.Setenv(envir.DevboxLatestVersion, "0.4.8")

		buf := new(bytes.Buffer)
		CheckVersion(buf)
		if buf.String() != "" {
			t.Errorf("expected empty string, got %q", buf.String())
		}
	})
}
