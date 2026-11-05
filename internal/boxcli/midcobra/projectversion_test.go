package midcobra

import (
	"testing"

	"github.com/spf13/cobra"
)

func testRootCmd() *cobra.Command {
	noop := func(*cobra.Command, []string) {}
	withConfig := func(cmd *cobra.Command) *cobra.Command {
		cmd.Flags().StringP("config", "c", "/default", "")
		return cmd
	}

	root := &cobra.Command{Use: "devbox"}
	root.PersistentFlags().BoolP("quiet", "q", false, "")

	shell := withConfig(&cobra.Command{Use: "shell", Run: noop})
	shell.Flags().Bool("pure", false, "")
	shell.Flags().StringSlice("env", nil, "")
	run := withConfig(&cobra.Command{Use: "run", Run: noop})
	run.Flags().Bool("all-projects", false, "")
	global := &cobra.Command{Use: "global"}
	global.AddCommand(withConfig(&cobra.Command{Use: "shellenv", Run: noop}))
	root.AddCommand(shell, run, global, &cobra.Command{Use: "version", Run: noop})
	return root
}

func TestProjectConfigPath(t *testing.T) {
	tests := []struct {
		args     []string
		wantPath string
		wantOK   bool
	}{
		{args: []string{"shell"}, wantPath: "/default", wantOK: true},
		{args: []string{"shell", "--config", "/a"}, wantPath: "/a", wantOK: true},
		{args: []string{"shell", "--config=/a"}, wantPath: "/a", wantOK: true},
		{args: []string{"shell", "-c", "/a"}, wantPath: "/a", wantOK: true},
		{args: []string{"-q", "shell", "--pure", "--env", "A=1", "-c", "/a"}, wantPath: "/a", wantOK: true},
		{args: []string{"shell", "--unknown", "value", "-c", "/a"}, wantPath: "/a", wantOK: true},
		{args: []string{"run", "-c", "/a", "--", "script", "-c", "/b"}, wantPath: "/a", wantOK: true},
		{args: []string{"run", "--all-projects", "--", "script"}, wantOK: false},
		{args: []string{"global", "shellenv"}, wantOK: false},
		{args: []string{"version"}, wantOK: false},
		{args: []string{"nope"}, wantOK: false},
	}
	for _, tc := range tests {
		gotPath, gotOK := projectConfigPath(testRootCmd(), tc.args)
		if gotPath != tc.wantPath || gotOK != tc.wantOK {
			t.Errorf("projectConfigPath(%q) = %q, %t; want %q, %t",
				tc.args, gotPath, gotOK, tc.wantPath, tc.wantOK)
		}
	}
}

// TestProjectConfigPathLeavesFlagsUnset checks that the middleware doesn't
// parse the command's own flags, which cobra parses later.
func TestProjectConfigPathLeavesFlagsUnset(t *testing.T) {
	root := testRootCmd()
	projectConfigPath(root, []string{"shell", "-c", "/a", "--env", "A=1"})

	shell, _, err := root.Find([]string{"shell"})
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"config", "env"} {
		if shell.Flag(name).Changed {
			t.Errorf("flag --%s was changed", name)
		}
	}
}
