// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"math"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type Cmd struct {
	// DestinationAddr is a "hostname[:port]" that specifies the remote host
	// and port to connect to.
	DestinationAddr string

	// Username is the remote login name.
	Username string

	PathInVM       string
	ShellStartTime string // unix timestamp
}

func Command(user, dest string) *Cmd {
	return &Cmd{DestinationAddr: dest, Username: user}
}

func (c *Cmd) Shell(w io.Writer) error {
	cmd := c.cmd("-t")
	remoteCmd := fmt.Sprintf(
		`bash -l -c "start_devbox_shell.sh \"%s\" %s"`,
		c.PathInVM,
		c.ShellStartTime,
	)
	cmd.Args = append(cmd.Args, remoteCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = io.MultiWriter(os.Stdout, w)
	cmd.Stderr = io.MultiWriter(os.Stderr, w)
	return logCmdRun(cmd)
}

func (c *Cmd) ExecRemote(cmd string) ([]byte, error) {
	sshCmd := c.cmd("-T")
	sshCmd.Args = append(sshCmd.Args, cmd)

	var stdout, stderr bytes.Buffer
	sshCmd.Stdout = &stdout
	sshCmd.Stderr = &stderr

	err := logCmdRun(sshCmd)
	logCmdOutput(sshCmd, "stderr", stderr.Bytes())
	if err != nil {
		// Only log output if there was an error, otherwise we might log
		// a VM's private key.
		logCmdOutput(sshCmd, "stdout", stdout.Bytes())
		return nil, err
	}
	return stdout.Bytes(), nil
}

// ExecRemoteWithRetry runs the given command on the remote host, retrying
// with an exponential backoff if the command fails. maxWait is the maximum
// seconds we wait in between retries.
func (c *Cmd) ExecRemoteWithRetry(cmd string, retries, maxWait int) ([]byte, error) {
	var err error
	var stdout []byte
	for i := 0; i < (retries + 1); i++ {
		if stdout, err = c.ExecRemote(cmd); err == nil {
			break
		}
		wait := int(math.Min(float64(maxWait), math.Pow(2, float64(i))))
		slog.Debug("retrying ExecRemote", "err", err, "wait", wait)
		time.Sleep(time.Duration(wait) * time.Second)
	}
	return stdout, err
}

func (c *Cmd) cmd(sshArgs ...string) *exec.Cmd {
	host, port := splitHostPort(c.DestinationAddr)
	cmd := exec.Command("ssh", "-l", c.Username)
	if port != 0 && port != 22 {
		cmd.Args = append(cmd.Args, "-p", strconv.Itoa(port))
	}
	cmd.Args = append(cmd.Args, sshArgs...)
	cmd.Args = append(cmd.Args, host)
	return cmd
}

// splitHostPort is like net.SplitHostPort except it defaults to port 22 if the
// port in the address is missing or invalid.
func splitHostPort(addr string) (host string, port int) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return addr, 22
	}
	port, err = net.LookupPort("tcp", portStr)
	if err != nil {
		return host, 22
	}
	return host, port
}

func logCmdRun(cmd *exec.Cmd) error {
	// Use cmd.Start so we can log the pid. Don't bother writing errors to
	// the debug log since those will be printed anyway.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("openssh: start command %q: %w", cmd, err)
	}
	slog.Debug("openssh: started process", "pid", cmd.Process.Pid, "cmd", cmd)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("openssh: process %d with command %q: %w",
			cmd.Process.Pid, cmd, err)
	}
	slog.Debug("openssh: process exited", "pid", cmd.Process.Pid, "cmd", cmd, "code", 0)
	return nil
}

func logCmdOutput(cmd *exec.Cmd, stdstream string, out []byte) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		slog.Debug("openssh: process exited", "pid", cmd.Process.Pid, "cmd", cmd, "code", cmd.ProcessState.ExitCode(), stdstream, out)
		return
	}

	out = bytes.ReplaceAll(out, []byte{'\n'}, []byte{'\n', '\t'})
	max := 1 << 16 // 64 KiB
	if overflow := len(out) - max; overflow > 0 {
		out = bytes.TrimSpace(out[:max])
		if overflow == 1 {
			out = append(out, "...truncated 1 byte."...)
		} else {
			out = fmt.Appendf(out, "...truncated %d bytes.", overflow)
		}
	}
	slog.Debug("openssh: process exited", "pid", cmd.Process.Pid, "cmd", cmd, "code", cmd.ProcessState.ExitCode(), stdstream, out)
}

type ControlSocket struct {
	Path string
	Host string
}

func DevboxControlSockets() []ControlSocket {
	socketsDir, err := devboxSocketsDir()
	if err != nil {
		return nil
	}

	// Look through whatever entries we got, even if there was an error.
	entries, _ := os.ReadDir(socketsDir)
	sockets := make([]ControlSocket, 0, len(entries))
	for _, entry := range entries {
		isSocket := (entry.Type() & fs.ModeSocket) == fs.ModeSocket
		if isSocket {
			sockets = append(sockets, ControlSocket{
				Path: filepath.Join(socketsDir, entry.Name()),

				// Right now the host is just the name, but this
				// will need to be updated if ControlPath in
				// sshconfig.tmpl ever changes.
				Host: entry.Name(),
			})
		}
	}
	return sockets
}
