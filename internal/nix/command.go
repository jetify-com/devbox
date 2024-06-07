package nix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

type cmd struct {
	Args cmdArgs
	Env  []string

	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	execCmd     *exec.Cmd
	execCmdOnce sync.Once
}

func command(args ...any) *cmd {
	cmd := &cmd{
		Args: append(cmdArgs{
			"nix",
			"--extra-experimental-features", "ca-derivations",
			"--option", "experimental-features", "nix-command flakes fetch-closure",
		}, args...),
	}
	return cmd
}

func (c *cmd) CombinedOutput(ctx context.Context) ([]byte, error) {
	out, err := c.initExecCommand(ctx).CombinedOutput()
	return out, c.error(ctx, err)
}

func (c *cmd) Output(ctx context.Context) ([]byte, error) {
	out, err := c.initExecCommand(ctx).Output()
	return out, c.error(ctx, err)
}

func (c *cmd) Run(ctx context.Context) error {
	return c.error(ctx, c.initExecCommand(ctx).Run())
}

func (c *cmd) String() string {
	return c.Args.String()
}

func (c *cmd) initExecCommand(ctx context.Context) *exec.Cmd {
	c.execCmdOnce.Do(func() {
		args := c.Args.StringSlice()
		c.execCmd = exec.CommandContext(ctx, args[0], args[1:]...)
		c.execCmd.Env = c.Env
		c.execCmd.Stdin = c.Stdin
		c.execCmd.Stdout = c.Stdout
		c.execCmd.Stderr = c.Stderr

		c.execCmd.Cancel = func() error {
			// Try to let Nix exit gracefully by sending an
			// interrupt instead of the default behavior of killing
			// it.
			err := c.execCmd.Process.Signal(os.Interrupt)
			if errors.Is(err, os.ErrProcessDone) {
				// Nix already exited; execCmd.Wait will use the
				// exit code.
				return err
			}
			if err != nil {
				// We failed to send SIGINT, so kill the process
				// instead.
				//
				// - If Nix already exited, Kill will return
				//   os.ErrProcessDone and execCmd.Wait will use
				//   the exit code.
				// - Otherwise, execCmd.Wait will always return
				//   an error.
				return c.execCmd.Process.Kill()
			}

			// We sent the SIGINT successfully. It's still possible
			// for Nix to exit successfully, so return
			// os.ErrProcessDone so that execCmd.Wait uses the exit
			// code instead of ctx.Err.
			return os.ErrProcessDone
		}
		// Kill Nix if it doesn't exit within 15 seconds of Devbox
		// sending an interrupt.
		c.execCmd.WaitDelay = 15 * time.Second
	})
	return c.execCmd
}

func (c *cmd) error(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}

	cmdErr := &cmdError{err: err}
	if errors.Is(err, exec.ErrNotFound) {
		cmdErr.msg = fmt.Sprintf("nix: %s not found in $PATH", c.Args[0])
	}

	switch {
	case errors.Is(ctx.Err(), context.Canceled):
		cmdErr.msg = "nix: command canceled"
	case errors.Is(ctx.Err(), context.DeadlineExceeded):
		cmdErr.msg = "nix: command timed out"
	default:
		cmdErr.msg = "nix: command error"
	}
	cmdErr.msg += ": " + c.String()

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if stderr := c.stderrExcerpt(exitErr.Stderr); len(stderr) != 0 {
			cmdErr.msg += ": " + stderr
		}
		if exitErr.Exited() {
			cmdErr.msg += fmt.Sprintf(": exit code %d", exitErr.ExitCode())
			return cmdErr
		}
		if stat, ok := exitErr.Sys().(syscall.WaitStatus); ok && stat.Signaled() {
			cmdErr.msg += fmt.Sprintf(": exit due to signal %d (%[1]s)", stat.Signal())
			return cmdErr
		}
	}

	if !errors.Is(err, ctx.Err()) {
		cmdErr.msg += ": " + err.Error()
	}
	return cmdErr
}

func (*cmd) stderrExcerpt(stderr []byte) string {
	stderr = bytes.TrimSpace(stderr)
	if len(stderr) == 0 {
		return ""
	}

	lines := bytes.Split(stderr, []byte("\n"))
	slices.Reverse(lines)
	for _, line := range lines {
		line = bytes.TrimSpace(line)
		after, found := bytes.CutPrefix(line, []byte("error: "))
		if !found {
			continue
		}
		after = bytes.TrimSpace(after)
		if len(after) == 0 {
			continue
		}
		stderr = after
		break

	}

	excerpt := string(stderr)
	if !strconv.CanBackquote(excerpt) {
		quoted := strconv.Quote(excerpt)
		excerpt = quoted[1 : len(quoted)-1]
	}
	return excerpt
}

type cmdArgs []any

func appendArgs[E any](args cmdArgs, new []E) cmdArgs {
	for _, elem := range new {
		args = append(args, elem)
	}
	return args
}

func (c cmdArgs) StringSlice() []string {
	s := make([]string, len(c))
	for i := range c {
		s[i] = fmt.Sprint(c[i])
	}
	return s
}

func (c cmdArgs) String() string {
	if len(c) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	c.writeQuoted(sb, fmt.Sprint(c[0]))
	if len(c) == 1 {
		return sb.String()
	}

	for _, arg := range c[1:] {
		sb.WriteByte(' ')
		c.writeQuoted(sb, fmt.Sprint(arg))
	}
	return sb.String()
}

func (cmdArgs) writeQuoted(dst *strings.Builder, str string) {
	needsQuote := strings.ContainsAny(str, ";\"'()$|&><` \t\r\n\\#{~*?[=")
	if !needsQuote {
		dst.WriteString(str)
		return
	}

	canSingleQuote := !strings.Contains(str, "'")
	if canSingleQuote {
		dst.WriteByte('\'')
		dst.WriteString(str)
		dst.WriteByte('\'')
		return
	}

	dst.WriteByte('"')
	for _, r := range str {
		switch r {
		// Special characters inside double quotes:
		// https://pubs.opengroup.org/onlinepubs/009604499/utilities/xcu_chap02.html#tag_02_02_03
		case '$', '`', '"', '\\':
			dst.WriteRune('\\')
		}
		dst.WriteRune(r)
	}
	dst.WriteByte('"')
}

type cmdError struct {
	msg string
	err error
}

func (c *cmdError) Redact() string {
	return c.Error()
}

func (c *cmdError) Error() string {
	return c.msg
}

func (c *cmdError) Unwrap() error {
	return c.err
}

func allowUnfreeEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_UNFREE=1")
}

func allowInsecureEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_INSECURE=1")
}
