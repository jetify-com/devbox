package midcobra

import (
	"context"
	"testing"

	"github.com/spf13/cobra"
)

type recordDebugMiddleware struct {
	debugChanged bool
}

func (r *recordDebugMiddleware) preRun(cmd *cobra.Command, args []string) {
	r.debugChanged = cmd.PersistentFlags().Lookup("debug").Changed
}

func (r *recordDebugMiddleware) postRun(cmd *cobra.Command, args []string, runErr error) {}

// TestExecuteParsesPersistentFlagsAfterUnknownFlags ensures that root
// persistent flags are visible to middlewares even when they follow a
// subcommand-specific flag.
func TestExecuteParsesPersistentFlagsAfterUnknownFlags(t *testing.T) {
	root := &cobra.Command{Use: "devbox", Run: func(*cobra.Command, []string) {}, SilenceErrors: true, SilenceUsage: true}
	root.PersistentFlags().Bool("debug", false, "")
	list := &cobra.Command{Use: "list", Run: func(*cobra.Command, []string) {}}
	list.Flags().Bool("outdated", false, "")
	root.AddCommand(list)

	rec := &recordDebugMiddleware{}
	ex := New(root)
	ex.AddMiddleware(rec)
	if code := ex.Execute(context.Background(), []string{"list", "--outdated", "--debug"}); code != 0 {
		t.Fatalf("Execute() = %d, want 0", code)
	}
	if !rec.debugChanged {
		t.Error("--debug after an unknown flag was not visible to middleware")
	}

	// Unknown flags must still be rejected when the root itself is executed.
	if code := New(root).Execute(context.Background(), []string{"--outdated"}); code == 0 {
		t.Error("Execute() with an unknown root flag = 0, want non-zero")
	}
}

func TestSkipTelemetry(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	if telemetrySkipped(cmd) {
		t.Error("telemetry skipped before calling SkipTelemetry")
	}
	SkipTelemetry(cmd)
	if !telemetrySkipped(cmd) {
		t.Error("telemetry not skipped after calling SkipTelemetry")
	}
}
