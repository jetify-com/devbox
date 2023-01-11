// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"bytes"
	"fmt"
	"io/fs"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"

	"go.jetpack.io/devbox/internal/debug"
)

type Client struct {
	Username string
	Addr     string
	PathInVM string
}

func (c *Client) Shell() error {
	cmd := c.cmd("-t")
	remoteCmd := fmt.Sprintf(`bash -l -c "start_devbox_shell.sh \"%s\""`, c.PathInVM)
	cmd.Args = append(cmd.Args, remoteCmd)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return logCmdRun(cmd)
}

func (c *Client) Exec(remoteCmd string) ([]byte, error) {
	sshCmd := c.cmd()
	sshCmd.Args = append(sshCmd.Args, remoteCmd)

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

func (c *Client) cmd(sshArgs ...string) *exec.Cmd {
	host, port := splitHostPort(c.Addr)
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Args = append(cmd.Args, destination(c.Username, host))

	// Add any necessary flags:
	if port != 0 && port != 22 {
		cmd.Args = append(cmd.Args, "-p", strconv.Itoa(port))
	}

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

func destination(username, hostname string) string {
	result := hostname
	if username != "" {
		result = username + "@" + result
	}

	return result
}

func logCmdRun(cmd *exec.Cmd) error {
	// Use cmd.Start so we can log the pid. Don't bother writing errors to
	// the debug log since those will be printed anyway.
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("openssh: start command %q: %w", cmd, err)
	}
	debug.Log("openssh: started process %d with command %q", cmd.Process.Pid, cmd)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("openssh: process %d with command %q: %w",
			cmd.Process.Pid, cmd, err)
	}
	debug.Log("openssh: process %d with command %q: exit status 0", cmd.Process.Pid, cmd)
	return nil
}

func logCmdOutput(cmd *exec.Cmd, stdstream string, out []byte) {
	out = bytes.TrimSpace(out)
	if len(out) == 0 {
		debug.Log("openssh: process %d with command %q: exit status %d: %s is empty",
			cmd.Process.Pid, cmd, cmd.ProcessState.ExitCode(), stdstream)
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
	debug.Log("openssh: process %d with command %q: exit status %d: %s text:\n\t%s",
		cmd.Process.Pid, cmd, cmd.ProcessState.ExitCode(), stdstream, out)
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
