// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package midcobra

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestDebugMiddlewarePostRunLogsErrorOnce(t *testing.T) {
	exitErr := exec.Command("sh", "-c", "exit 3").Run()
	var execExitErr *exec.ExitError
	if !errors.As(exitErr, &execExitErr) {
		t.Fatalf("got error %v, want an *exec.ExitError", exitErr)
	}

	tests := []struct {
		name       string
		err        error
		wantStderr bool
	}{
		{"exec.ExitError", fmt.Errorf("running nix: %w", exitErr), true},
		{"other error", errors.New("boom"), false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var logs bytes.Buffer
			defer slog.SetDefault(slog.Default())
			slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))

			cmd := &cobra.Command{}
			cmd.SetErr(io.Discard)
			(&DebugMiddleware{}).postRun(cmd, nil, test.err)

			out := logs.String()
			if got := strings.Count(out, `msg="command error"`); got != 1 {
				t.Errorf("logged %d command errors, want 1:\n%s", got, out)
			}
			if got := strings.Contains(out, "stderr="); got != test.wantStderr {
				t.Errorf("log contains stderr = %v, want %v:\n%s", got, test.wantStderr, out)
			}
		})
	}
}
