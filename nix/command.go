package nix

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// Cmd is an external command that invokes a [*Nix] executable. It provides
// improved error messages, graceful cancellation, and debug logging via
// [log/slog]. Although it's possible to initialize a Cmd directly, calling the
// [Command] function or [Nix.Command] method is more typical.
//
// Most methods and fields correspond to their [exec.Cmd] equivalent. See its
// documentation for more details.
type Cmd struct {
	// Path is the absolute path to the nix executable. It is the only
	// mandatory field and must not be empty.
	Path string

	// Args are the command line arguments, including the command name in
	// Args[0]. Run formats each argument using [fmt.Sprint] before passing
	// them to Nix.
	Args Args

	Env    []string
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer

	// Logger emits debug logs when the command starts and exits. If nil, it
	// defaults to [slog.Default].
	Logger *slog.Logger

	execCmd *exec.Cmd
	err     error
	dur     time.Duration
}

// Command creates an arbitrary Nix command that uses the Path, ExtraArgs,
// Logger and other defaults from n.
func (n *Nix) Command(args ...any) *Cmd {
	cmd := &Cmd{
		Args:   make(Args, 1, 1+len(n.ExtraArgs)+len(args)),
		Logger: n.logger(),
	}
	cmd.Path, cmd.err = n.resolvePath()

	if n.Path == "" {
		cmd.Args[0] = "nix" // resolved from $PATH
	} else {
		cmd.Args[0] = n.Path // explicitly set
	}
	cmd.Args = append(cmd.Args, n.ExtraArgs...)
	cmd.Args = append(cmd.Args, args...)
	return cmd
}

func (c *Cmd) CombinedOutput(ctx context.Context) ([]byte, error) {
	defer c.logRunFunc(ctx)()

	start := time.Now()
	out, err := c.initExecCommand(ctx).CombinedOutput()
	c.dur = time.Since(start)

	c.err = c.error(ctx, err)
	return out, c.err
}

func (c *Cmd) Output(ctx context.Context) ([]byte, error) {
	defer c.logRunFunc(ctx)()

	start := time.Now()
	out, err := c.initExecCommand(ctx).Output()
	c.dur = time.Since(start)

	c.err = c.error(ctx, err)
	return out, c.err
}

func (c *Cmd) Run(ctx context.Context) error {
	defer c.logRunFunc(ctx)()

	start := time.Now()
	err := c.initExecCommand(ctx).Run()
	c.dur = time.Since(start)

	c.err = c.error(ctx, err)
	return c.err
}

func (c *Cmd) LogValue() slog.Value {
	attrs := []slog.Attr{
		slog.Any("args", c.Args),
	}
	if c.execCmd == nil {
		return slog.GroupValue(attrs...)
	}
	attrs = append(attrs, slog.String("path", c.execCmd.Path))

	var exitErr *exec.ExitError
	if errors.As(c.err, &exitErr) {
		stderr := c.stderrExcerpt(exitErr.Stderr)
		if len(stderr) != 0 {
			attrs = append(attrs, slog.String("stderr", stderr))
		}
	}
	if proc := c.execCmd.Process; proc != nil {
		attrs = append(attrs, slog.Int("pid", proc.Pid))
	}
	if procState := c.execCmd.ProcessState; procState != nil {
		if procState.Exited() {
			attrs = append(attrs, slog.Int("code", procState.ExitCode()))
		}
		if status, ok := procState.Sys().(syscall.WaitStatus); ok && status.Signaled() {
			if status.Signaled() {
				attrs = append(attrs, slog.String("signal", status.Signal().String()))
			}
		}
	}
	if c.dur != 0 {
		attrs = append(attrs, slog.Duration("dur", c.dur))
	}
	return slog.GroupValue(attrs...)
}

// String returns c as a shell-quoted string.
func (c *Cmd) String() string {
	return c.Args.String()
}

func (c *Cmd) initExecCommand(ctx context.Context) *exec.Cmd {
	if c.execCmd != nil {
		return c.execCmd
	}

	c.execCmd = exec.CommandContext(ctx, c.Path)
	c.execCmd.Path = c.Path
	c.execCmd.Args = c.Args.StringSlice()
	c.execCmd.Env = c.Env
	c.execCmd.Stdin = c.Stdin
	c.execCmd.Stdout = c.Stdout
	c.execCmd.Stderr = c.Stderr

	c.execCmd.Cancel = func() error {
		// Try to let Nix exit gracefully by sending an interrupt
		// instead of the default behavior of killing it.
		c.logger().DebugContext(ctx, "sending interrupt to nix process", slog.Group("cmd",
			"args", c.Args,
			"path", c.execCmd.Path,
			"pid", c.execCmd.Process.Pid,
		))
		err := c.execCmd.Process.Signal(os.Interrupt)
		if errors.Is(err, os.ErrProcessDone) {
			// Nix already exited; execCmd.Wait will use the exit
			// code.
			return err
		}
		if err != nil {
			// We failed to send SIGINT, so kill the process
			// instead.
			//
			// - If Nix already exited, Kill will return
			//   os.ErrProcessDone and execCmd.Wait will use
			//   the exit code.
			// - Otherwise, execCmd.Wait will always return an
			//   error.
			c.logger().DebugContext(ctx, "error interrupting nix process, attempting to kill",
				"err", err, slog.Group("cmd",
					"args", c.Args,
					"path", c.execCmd.Path,
					"pid", c.execCmd.Process.Pid,
				))
			return c.execCmd.Process.Kill()
		}

		// We sent the SIGINT successfully. It's still possible for Nix
		// to exit successfully, so return os.ErrProcessDone so that
		// execCmd.Wait uses the exit code instead of ctx.Err.
		return os.ErrProcessDone
	}
	// Kill Nix if it doesn't exit within 15 seconds of Devbox sending an
	// interrupt.
	c.execCmd.WaitDelay = 15 * time.Second
	return c.execCmd
}

func (c *Cmd) error(ctx context.Context, err error) error {
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

func (*Cmd) stderrExcerpt(stderr []byte) string {
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

func (c *Cmd) logger() *slog.Logger {
	if c.Logger == nil {
		return slog.Default()
	}
	return c.Logger
}

// logRunFunc logs the start and exit of c.execCmd. It adjusts the source
// attribute of the log record to point to the caller of c.CombinedOutput,
// c.Output, or c.Run. This assumes a specific stack depth, so do not call
// logRunFunc from other methods or functions.
func (c *Cmd) logRunFunc(ctx context.Context) func() {
	logger := c.logger()
	if !logger.Enabled(ctx, slog.LevelDebug) {
		return func() {}
	}

	var pcs [1]uintptr
	runtime.Callers(3, pcs[:]) // skip Callers, logRunFunc, CombinedOutput/Output/Run
	r := slog.NewRecord(time.Now(), slog.LevelDebug, "nix command starting", pcs[0])
	r.Add("cmd", c)
	_ = logger.Handler().Handle(ctx, r)

	return func() {
		r := slog.NewRecord(time.Now(), slog.LevelDebug, "nix command exited", pcs[0])
		r.Add("cmd", c)
		_ = logger.Handler().Handle(ctx, r)
	}
}

// Args is a slice of [Cmd] arguments.
type Args []any

// StringSlice formats each argument using [fmt.Sprint].
func (a Args) StringSlice() []string {
	s := make([]string, len(a))
	for i := range a {
		s[i] = fmt.Sprint(a[i])
	}
	return s
}

// String returns the arguments as a shell command, quoting arguments with
// spaces.
func (a Args) String() string {
	if len(a) == 0 {
		return ""
	}

	sb := &strings.Builder{}
	a.writeQuoted(sb, fmt.Sprint(a[0]))
	if len(a) == 1 {
		return sb.String()
	}

	for _, arg := range a[1:] {
		sb.WriteByte(' ')
		a.writeQuoted(sb, fmt.Sprint(arg))
	}
	return sb.String()
}

func (Args) writeQuoted(dst *strings.Builder, str string) {
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
