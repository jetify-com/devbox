// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/creack/pty"

	"go.jetify.com/devbox/internal/boxcli/usererr"
)

const runScriptHelperEnv = "DEVBOX_TEST_RUN_SCRIPT_HELPER"

// TestRunScriptHelper plays the part of `devbox run` for
// TestRunScriptWaitsForCommandOnCtrlC, which runs it in a terminal.
func TestRunScriptHelper(t *testing.T) {
	if os.Getenv(runScriptHelperEnv) == "" {
		t.Skip("helper process for TestRunScriptWaitsForCommandOnCtrlC")
	}
	script := `trap 'echo command: shutting down; sleep 0.5; echo command: shutdown complete; exit 3' INT
echo command: started
while true; do sleep 0.1; done`
	err := RunScript(t.TempDir(), script, map[string]string{"PATH": os.Getenv("PATH")})
	var exitErr *usererr.ExitError
	if errors.As(err, &exitErr) {
		fmt.Printf("RunScript returned exit code %d\n", exitErr.ExitCode())
	} else {
		fmt.Printf("RunScript returned %v\n", err)
	}
	os.Exit(0)
}

// TestRunScriptWaitsForCommandOnCtrlC checks that pressing Ctrl-C doesn't kill
// devbox while the command is still shutting down (#3026).
func TestRunScriptWaitsForCommandOnCtrlC(t *testing.T) {
	cmd := exec.Command(os.Args[0], "-test.run=^TestRunScriptHelper$")
	cmd.Env = append(os.Environ(), runScriptHelperEnv+"=1")
	terminal, err := pty.Start(cmd) // the helper's controlling terminal
	if err != nil {
		t.Fatal(err)
	}
	defer terminal.Close()

	var outMu sync.Mutex
	var out bytes.Buffer
	go func() {
		buf := make([]byte, 1024)
		for {
			n, err := terminal.Read(buf)
			outMu.Lock()
			out.Write(buf[:n])
			outMu.Unlock()
			if err != nil {
				return
			}
		}
	}()
	output := func() string {
		outMu.Lock()
		defer outMu.Unlock()
		return out.String()
	}

	deadline := time.Now().Add(10 * time.Second)
	for !strings.Contains(output(), "command: started") {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for the command to start, output:\n%s", output())
		}
		time.Sleep(10 * time.Millisecond)
	}
	if _, err := io.WriteString(terminal, "\x03"); err != nil { // Ctrl-C
		t.Fatal(err)
	}

	waitErr := make(chan error, 1)
	go func() { waitErr <- cmd.Wait() }()
	select {
	case err := <-waitErr:
		if err != nil {
			t.Fatalf("devbox didn't exit cleanly: %v, output:\n%s", err, output())
		}
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for devbox to exit, output:\n%s", output())
	}

	// Give the reader a moment to drain what's left in the terminal.
	time.Sleep(100 * time.Millisecond)
	got := output()
	complete := strings.Index(got, "command: shutdown complete")
	returned := strings.Index(got, "RunScript returned exit code 3")
	if complete == -1 || returned == -1 || returned < complete {
		t.Errorf("RunScript should return the command's exit code after it shuts down, output:\n%s", got)
	}
}
