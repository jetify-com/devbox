// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package vercheck

import (
	"bytes"
	"os"
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
