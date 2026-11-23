// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/shellgen"
)

// TestConcurrentRunScriptFiles mimics many `devbox run -- <cmd>` invocations
// in one project: each rewrites the scripts and then executes the shared
// .cmd.sh. None of them should see a missing, empty, or non-executable script.
func TestConcurrentRunScriptFiles(t *testing.T) {
	dir := t.TempDir()
	config := `{"shell": {"init_hook": ["true"], "scripts": {"hello": "echo hello"}}}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "devbox.json"), []byte(config), 0o644))
	box, err := Open(&devopt.Opts{Dir: dir, Stderr: os.Stderr})
	require.NoError(t, err)

	cmdBody, err := shellgen.ScriptBody(box, "eval $DEVBOX_RUN_CMD\n")
	require.NoError(t, err)
	script := shellgen.ScriptPath(dir, shellgen.ArbitraryCmdFilename)

	const workers, runs = 8, 20
	errs := make(chan error, workers*runs)
	var group sync.WaitGroup
	for worker := range workers {
		group.Go(func() {
			for run := range runs {
				if err := shellgen.WriteScriptsToFiles(box); err != nil {
					errs <- err
					continue
				}
				if err := shellgen.WriteScriptFile(box, shellgen.ArbitraryCmdFilename, cmdBody); err != nil {
					errs <- err
					continue
				}
				want := fmt.Sprintf("hello-%d-%d", worker, run)
				cmd := exec.Command("sh", "-c", script)
				cmd.Env = append(os.Environ(), "DEVBOX_RUN_CMD=echo "+want)
				out, err := cmd.CombinedOutput()
				if err != nil || string(out) != want+"\n" {
					errs <- fmt.Errorf("run %s: err=%v, output=%q", want, err, out)
				}
			}
		})
	}
	group.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	// Concurrent writers shouldn't leave temp files behind.
	entries, err := os.ReadDir(filepath.Dir(script))
	require.NoError(t, err)
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.ElementsMatch(t, []string{".cmd.sh", ".hooks.sh", "hello.sh"}, names)
}
