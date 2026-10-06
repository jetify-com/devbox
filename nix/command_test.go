package nix

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// fakeNix writes a script that pretends to be Nix. Each invocation prints
// the next entry in stderrs to stderr and exits with an error, until it runs
// out of entries and succeeds. It returns the script path and a function that
// reports how many times the script ran.
func fakeNix(t *testing.T, stderrs ...string) (path string, runs func() int) {
	t.Helper()
	retryDelay = 0
	t.Cleanup(func() { retryDelay = 2 * time.Second })

	dir := t.TempDir()
	countFile := filepath.Join(dir, "count")
	script := "#!/bin/sh\n" +
		"n=$(cat " + countFile + " 2>/dev/null || echo 0)\n" +
		"echo $((n + 1)) > " + countFile + "\n" +
		"case $n in\n"
	for i, stderr := range stderrs {
		script += strconv.Itoa(i) + ") echo '" + stderr + "' >&2; exit 1 ;;\n"
	}
	script += "esac\necho ok\n"

	path = filepath.Join(dir, "nix")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	return path, func() int {
		// Don't fail the test here, it can be called from another
		// goroutine and before the script has run.
		b, _ := os.ReadFile(countFile)
		n, _ := strconv.Atoi(strings.TrimSpace(string(b)))
		return n
	}
}

const truncatedTarErr = "error: cannot read file from tarball: Truncated tar archive detected while reading data"

func TestCmdOutputRetriesTransientError(t *testing.T) {
	path, runs := fakeNix(t, truncatedTarErr)
	cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3}

	out, err := cmd.Output(t.Context())
	if err != nil {
		t.Fatalf("got error: %v", err)
	}
	if got, want := strings.TrimSpace(string(out)), "ok"; got != want {
		t.Errorf("got output %q, want %q", got, want)
	}
	if got, want := runs(), 2; got != want {
		t.Errorf("got %d runs, want %d", got, want)
	}
}

func TestCmdRunRetriesTransientErrorWithStderr(t *testing.T) {
	path, runs := fakeNix(t, "error: unable to download 'https://github.com/x': HTTP error 502")
	stderr := &bytes.Buffer{}
	cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3, Stderr: stderr}

	if err := cmd.Run(t.Context()); err != nil {
		t.Fatalf("got error: %v", err)
	}
	if got, want := runs(), 2; got != want {
		t.Errorf("got %d runs, want %d", got, want)
	}
	if !strings.Contains(stderr.String(), "HTTP error 502") {
		t.Errorf("stderr doesn't contain Nix's original error:\n%s", stderr)
	}
	if !strings.Contains(stderr.String(), "retrying") {
		t.Errorf("stderr doesn't contain a retry message:\n%s", stderr)
	}
}

func TestCmdCombinedOutputRetriesTransientError(t *testing.T) {
	path, runs := fakeNix(t, truncatedTarErr)
	cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3}

	if _, err := cmd.CombinedOutput(t.Context()); err != nil {
		t.Fatalf("got error: %v", err)
	}
	if got, want := runs(), 2; got != want {
		t.Errorf("got %d runs, want %d", got, want)
	}
}

func TestCmdGivesUpAfterMaxAttempts(t *testing.T) {
	path, runs := fakeNix(t, truncatedTarErr, truncatedTarErr, truncatedTarErr)
	cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3}

	_, err := cmd.Output(t.Context())
	if err == nil {
		t.Fatal("got nil error, want error after max attempts")
	}
	if !strings.Contains(err.Error(), "Truncated tar archive") {
		t.Errorf("got error %q, want it to contain Nix's error", err)
	}
	if got, want := runs(), 3; got != want {
		t.Errorf("got %d runs, want %d", got, want)
	}
}

func TestCmdDoesNotRetry(t *testing.T) {
	tests := []struct {
		name   string
		stderr string
		modify func(*Cmd)
	}{
		{
			name:   "NonTransientError",
			stderr: "error: flake 'path:/x' does not provide attribute 'packages.aarch64-darwin.foo'",
		},
		{
			name:   "RateLimit",
			stderr: "error: unable to download 'https://api.github.com/x': HTTP error 403",
		},
		{
			name:   "RetriesDisabled",
			stderr: truncatedTarErr,
			modify: func(c *Cmd) { c.MaxAttempts = 0 },
		},
		{
			name:   "NonFileStdin",
			stderr: truncatedTarErr,
			modify: func(c *Cmd) { c.Stdin = strings.NewReader("input") },
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path, runs := fakeNix(t, test.stderr)
			cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3}
			if test.modify != nil {
				test.modify(cmd)
			}

			if _, err := cmd.Output(t.Context()); err == nil {
				t.Fatal("got nil error, want error")
			}
			if got, want := runs(), 1; got != want {
				t.Errorf("got %d runs, want %d", got, want)
			}
		})
	}
}

func TestCmdStopsRetryingWhenCanceled(t *testing.T) {
	path, runs := fakeNix(t, truncatedTarErr)
	retryDelay = time.Hour

	ctx, cancel := context.WithCancel(t.Context())
	cmd := &Cmd{Path: path, Args: Args{path}, MaxAttempts: 3, Stderr: &bytes.Buffer{}}
	go func() {
		for runs() == 0 {
			time.Sleep(10 * time.Millisecond)
		}
		cancel()
	}()

	if err := cmd.Run(ctx); err == nil {
		t.Fatal("got nil error, want error")
	}
	if got, want := runs(), 1; got != want {
		t.Errorf("got %d runs, want %d", got, want)
	}
}

func TestTailWriter(t *testing.T) {
	w := &tailWriter{}
	_, _ = w.Write(bytes.Repeat([]byte("x"), 10<<10))
	_, _ = w.Write([]byte(truncatedTarErr))
	if got, want := len(w.buf), 8<<10; got != want {
		t.Errorf("got len %d, want %d", got, want)
	}
	if !bytes.HasSuffix(w.buf, []byte(truncatedTarErr)) {
		t.Error("tail doesn't end with the last write")
	}
}

func TestTailWriterSmallWrites(t *testing.T) {
	tail := &tailWriter{}
	line := []byte("copying path '/nix/store/xxx' from 'https://cache.nixos.org'\n")
	for range 1000 {
		_, _ = tail.Write(line)
	}
	_, _ = tail.Write([]byte(truncatedTarErr))
	if got, want := len(tail.buf), 8<<10; got != want {
		t.Errorf("got len %d, want %d", got, want)
	}
	if got, limit := cap(tail.buf), 16<<10; got > limit {
		t.Errorf("got cap %d, want <= %d", got, limit)
	}
	if !bytes.HasSuffix(tail.buf, []byte(truncatedTarErr)) {
		t.Error("tail doesn't end with the last write")
	}
}
