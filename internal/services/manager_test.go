// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package services

import (
	"testing"

	"go.jetify.com/devbox/internal/envir"
)

// TestGetProcessManagerPortReleasesLock verifies that GetProcessManagerPort
// releases the global config file lock, so that it can be called repeatedly
// in the same process (e.g. `devbox services start a b`).
func TestGetProcessManagerPortReleasesLock(t *testing.T) {
	t.Setenv(envir.XDGDataHome, t.TempDir())

	projectDir := t.TempDir()
	configFile, err := openGlobalConfigFile()
	if err != nil {
		t.Fatal(err)
	}
	config := newGlobalProcessComposeConfig()
	config.Instances[projectDir] = instance{Pid: 1, Port: 1234}
	if err := writeGlobalProcessComposeJSON(config, configFile); err != nil {
		t.Fatal(err)
	}
	configFile.Close()

	for range 2 {
		port, err := GetProcessManagerPort(projectDir)
		if err != nil {
			t.Fatalf("GetProcessManagerPort() error = %v", err)
		}
		if port != 1234 {
			t.Errorf("GetProcessManagerPort() = %d, want 1234", port)
		}
	}
}
