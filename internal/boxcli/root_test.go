// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"io"
	"testing"

	"github.com/spf13/cobra"
)

// TestRootPersistentPreRunRunsForSubcommandsWithOwnHook ensures the root
// command's PersistentPreRun (e.g. --quiet handling) still runs for
// subcommands whose parent declares its own PersistentPreRunE.
func TestRootPersistentPreRunRunsForSubcommandsWithOwnHook(t *testing.T) {
	root := RootCmd()
	generate, _, err := root.Find([]string{"generate"})
	if err != nil {
		t.Fatal(err)
	}
	parentHookRan := false
	generate.PersistentPreRunE = func(*cobra.Command, []string) error {
		parentHookRan = true
		return nil
	}
	quieted := false
	generate.AddCommand(&cobra.Command{
		Use: "test-hooks",
		RunE: func(cmd *cobra.Command, args []string) error {
			quieted = cmd.ErrOrStderr() == io.Discard
			return nil
		},
	})

	root.SetArgs([]string{"generate", "test-hooks", "--quiet"})
	if err := root.Execute(); err != nil {
		t.Fatal(err)
	}
	if !parentHookRan {
		t.Error("generate's PersistentPreRunE did not run")
	}
	if !quieted {
		t.Error("root's PersistentPreRun did not run: --quiet was ignored")
	}
}
