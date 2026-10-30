// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package usererr

import (
	"errors"
	"os/exec"
	"testing"
)

func TestExitErrorExitCode(t *testing.T) {
	tests := map[string]struct {
		script          string
		wantCode        int
		wantInterrupted bool
	}{
		"exit status":       {script: "exit 3", wantCode: 3},
		"killed by SIGINT":  {script: "kill -INT $$", wantCode: 130, wantInterrupted: true},
		"killed by SIGTERM": {script: "kill -TERM $$", wantCode: 143},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			err := NewExecError(exec.Command("sh", "-c", test.script).Run())
			var exitErr *ExitError
			if !errors.As(err, &exitErr) {
				t.Fatalf("got error %v (%T), want *ExitError", err, err)
			}
			if got := exitErr.ExitCode(); got != test.wantCode {
				t.Errorf("got exit code %d, want %d", got, test.wantCode)
			}
			if got := exitErr.Interrupted(); got != test.wantInterrupted {
				t.Errorf("got Interrupted() = %t, want %t", got, test.wantInterrupted)
			}
		})
	}
}
