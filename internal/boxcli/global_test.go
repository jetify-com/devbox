// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import "testing"

func TestEnsureGlobalConfigCachesPath(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", t.TempDir())
	globalConfigPath = ""
	t.Cleanup(func() { globalConfigPath = "" })

	path, err := ensureGlobalConfig()
	if err != nil {
		t.Fatal(err)
	}
	if globalConfigPath != path {
		t.Errorf("globalConfigPath = %q, want cached %q", globalConfigPath, path)
	}
}
