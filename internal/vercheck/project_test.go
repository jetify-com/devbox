// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/envir"
)

const testConfigPath = "/project/devbox.json"

// setupProjectVersionTest mocks the running devbox version and clears any
// environment that affects CheckProjectVersion.
func setupProjectVersionTest(t *testing.T, version string) {
	t.Helper()
	oldVersion, oldIsDev := currentDevboxVersion, isDevBuild
	t.Cleanup(func() { currentDevboxVersion, isDevBuild = oldVersion, oldIsDev })
	currentDevboxVersion = version
	isDevBuild = false

	t.Setenv(envir.DevboxVersionPolicy, "")
	t.Setenv(envir.DevboxUseVersion, "")
	t.Setenv(autoVersionEnvName, "")
	t.Setenv(envir.DevboxLatestVersion, "")
	t.Setenv(warnedEnvName, "")
	t.Cleanup(func() { os.Unsetenv(warnedEnvName) })
}

func TestCheckProjectVersionSatisfied(t *testing.T) {
	setupProjectVersionTest(t, "0.18.4")

	for _, version := range []string{"0.18.4", "v0.18.4", "^0.18.0", "~0.18.1", ">=0.17.0 <0.19.0", "0.18.x"} {
		t.Run(version, func(t *testing.T) {
			buf := new(bytes.Buffer)
			err := CheckProjectVersion(buf, testConfigPath, &configfile.DevboxVersion{
				Version:    version,
				OnMismatch: configfile.VersionPolicyError,
			})
			require.NoError(t, err)
			assert.Empty(t, buf.String())
		})
	}
}

func TestCheckProjectVersionNoConstraint(t *testing.T) {
	setupProjectVersionTest(t, "0.18.4")

	buf := new(bytes.Buffer)
	require.NoError(t, CheckProjectVersion(buf, testConfigPath, nil))
	assert.Empty(t, buf.String())
}

func TestCheckProjectVersionSkipsDevBuild(t *testing.T) {
	setupProjectVersionTest(t, "0.0.0-dev")
	isDevBuild = true

	buf := new(bytes.Buffer)
	err := CheckProjectVersion(buf, testConfigPath, &configfile.DevboxVersion{
		Version:    "0.18.4",
		OnMismatch: configfile.VersionPolicyError,
	})
	require.NoError(t, err)
	assert.Empty(t, buf.String())
}

func TestCheckProjectVersionWarn(t *testing.T) {
	setupProjectVersionTest(t, "0.17.2")
	t.Setenv(envir.DevboxLatestVersion, "0.18.4")
	required := &configfile.DevboxVersion{Version: "^0.18.0"}

	buf := new(bytes.Buffer)
	require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
	assert.Contains(t, buf.String(), "requires devbox ^0.18.0")
	assert.Contains(t, buf.String(), "you are running 0.17.2")
	assert.Contains(t, buf.String(), "devbox version update")

	// A nested devbox command for the same project doesn't warn again.
	buf.Reset()
	require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
	assert.Empty(t, buf.String())

	// A different project still warns.
	require.NoError(t, CheckProjectVersion(buf, "/other/devbox.json", required))
	assert.Contains(t, buf.String(), "requires devbox ^0.18.0")
}

func TestCheckProjectVersionUpdateSuggestion(t *testing.T) {
	required := &configfile.DevboxVersion{Version: ">=0.17.0 <0.19.0", OnMismatch: configfile.VersionPolicyError}

	t.Run("latest_satisfies", func(t *testing.T) {
		setupProjectVersionTest(t, "0.16.0")
		t.Setenv(envir.DevboxLatestVersion, "0.18.4")
		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		assert.ErrorContains(t, err, "devbox version update")
	})

	// Updating can't go backwards, so don't suggest it when the running
	// version is already past the constraint's upper bound.
	t.Run("too_new", func(t *testing.T) {
		setupProjectVersionTest(t, "0.20.0")
		t.Setenv(envir.DevboxLatestVersion, "0.20.0")
		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		require.ErrorContains(t, err, "DEVBOX_USE_VERSION")
		assert.NotContains(t, err.Error(), "devbox version update")
	})

	t.Run("latest_unknown", func(t *testing.T) {
		setupProjectVersionTest(t, "0.16.0")
		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		require.ErrorContains(t, err, "DEVBOX_USE_VERSION")
		assert.NotContains(t, err.Error(), "devbox version update")
	})
}

func TestCheckProjectVersionError(t *testing.T) {
	setupProjectVersionTest(t, "0.17.2")

	buf := new(bytes.Buffer)
	err := CheckProjectVersion(buf, testConfigPath, &configfile.DevboxVersion{
		Version:    "0.18.4",
		OnMismatch: configfile.VersionPolicyError,
	})
	require.Error(t, err)
	userErr, ok := usererr.Extract(err)
	require.True(t, ok, "expected a user error")
	assert.Contains(t, userErr.Error(), "requires devbox 0.18.4")
	assert.Contains(t, userErr.Error(), "DEVBOX_USE_VERSION=0.18.4")
	assert.Empty(t, buf.String())
}

func TestCheckProjectVersionEnvOverride(t *testing.T) {
	setupProjectVersionTest(t, "0.17.2")
	required := &configfile.DevboxVersion{Version: "0.18.4", OnMismatch: configfile.VersionPolicyError}

	t.Run("off", func(t *testing.T) {
		t.Setenv(envir.DevboxVersionPolicy, "off")
		buf := new(bytes.Buffer)
		require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
		assert.Empty(t, buf.String())
	})

	t.Run("warn", func(t *testing.T) {
		t.Setenv(envir.DevboxVersionPolicy, "warn")
		buf := new(bytes.Buffer)
		require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
		assert.Contains(t, buf.String(), "requires devbox 0.18.4")
	})

	t.Run("error", func(t *testing.T) {
		t.Setenv(envir.DevboxVersionPolicy, "error")
		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, &configfile.DevboxVersion{Version: "0.18.4"})
		assert.Error(t, err)
	})

	t.Run("invalid", func(t *testing.T) {
		t.Setenv(envir.DevboxVersionPolicy, "explode")
		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		assert.ErrorContains(t, err, envir.DevboxVersionPolicy)
	})
}

// mockExec replaces execFunc and records the call instead of replacing the
// process.
type mockExec struct {
	called bool
	argv0  string
	argv   []string
	envv   []string
}

func (m *mockExec) install(t *testing.T) {
	t.Helper()
	old := execFunc
	t.Cleanup(func() { execFunc = old })
	execFunc = func(argv0 string, argv, envv []string) error {
		m.called, m.argv0, m.argv, m.envv = true, argv0, argv, envv
		return nil
	}
}

func (m *mockExec) env(name string) []string {
	var values []string
	for _, kv := range m.envv {
		if k, v, _ := strings.Cut(kv, "="); k == name {
			values = append(values, v)
		}
	}
	return values
}

func TestCheckProjectVersionAuto(t *testing.T) {
	required := &configfile.DevboxVersion{Version: "v0.18.4", OnMismatch: configfile.VersionPolicyAuto}

	t.Run("switches_with_launcher", func(t *testing.T) {
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.LauncherPath, "/usr/local/bin/devbox")
		t.Setenv(envir.DevboxUseVersion, "")
		t.Setenv(autoVersionEnvName, "")
		exec := &mockExec{}
		exec.install(t)

		buf := new(bytes.Buffer)
		require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
		require.True(t, exec.called)
		assert.Equal(t, "/usr/local/bin/devbox", exec.argv0)
		assert.Equal(t, append([]string{"/usr/local/bin/devbox"}, os.Args[1:]...), exec.argv)
		// The existing empty values are replaced, not duplicated.
		assert.Equal(t, []string{"0.18.4"}, exec.env(envir.DevboxUseVersion))
		assert.Equal(t, []string{"0.18.4"}, exec.env(autoVersionEnvName))
		assert.Empty(t, buf.String())
	})

	t.Run("switches_again_for_a_different_project", func(t *testing.T) {
		// Inside a shell that auto switched to 0.17.2 for another project.
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.LauncherPath, "/usr/local/bin/devbox")
		t.Setenv(envir.DevboxUseVersion, "0.17.2")
		t.Setenv(autoVersionEnvName, "0.17.2")
		exec := &mockExec{}
		exec.install(t)

		require.NoError(t, CheckProjectVersion(new(bytes.Buffer), testConfigPath, required))
		require.True(t, exec.called)
		assert.Equal(t, []string{"0.18.4"}, exec.env(envir.DevboxUseVersion))
	})

	t.Run("no_launcher", func(t *testing.T) {
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.LauncherPath, "")
		exec := &mockExec{}
		exec.install(t)

		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		assert.ErrorContains(t, err, "wasn't started by the devbox launcher")
		assert.False(t, exec.called)
	})

	t.Run("launcher_did_not_switch", func(t *testing.T) {
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.LauncherPath, "/usr/local/bin/devbox")
		t.Setenv(envir.DevboxUseVersion, "0.18.4")
		t.Setenv(autoVersionEnvName, "0.18.4")
		exec := &mockExec{}
		exec.install(t)

		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, required)
		assert.ErrorContains(t, err, "launcher ran version 0.17.2 instead")
		assert.False(t, exec.called)
	})

	t.Run("user_set_version_wins", func(t *testing.T) {
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.LauncherPath, "/usr/local/bin/devbox")
		t.Setenv(envir.DevboxUseVersion, "0.17.2")
		t.Setenv(autoVersionEnvName, "")
		exec := &mockExec{}
		exec.install(t)

		buf := new(bytes.Buffer)
		require.NoError(t, CheckProjectVersion(buf, testConfigPath, required))
		assert.False(t, exec.called)
		assert.Contains(t, buf.String(), "DEVBOX_USE_VERSION=0.17.2 is set")
	})

	t.Run("env_override_with_range", func(t *testing.T) {
		setupProjectVersionTest(t, "0.17.2")
		t.Setenv(envir.DevboxVersionPolicy, "auto")
		t.Setenv(envir.LauncherPath, "/usr/local/bin/devbox")
		exec := &mockExec{}
		exec.install(t)

		err := CheckProjectVersion(new(bytes.Buffer), testConfigPath, &configfile.DevboxVersion{Version: "^0.18.0"})
		assert.ErrorContains(t, err, "not an exact version")
		assert.False(t, exec.called)
	})

	t.Run("satisfied", func(t *testing.T) {
		setupProjectVersionTest(t, "0.18.4")
		exec := &mockExec{}
		exec.install(t)

		require.NoError(t, CheckProjectVersion(new(bytes.Buffer), testConfigPath, required))
		assert.False(t, exec.called)
	})
}
