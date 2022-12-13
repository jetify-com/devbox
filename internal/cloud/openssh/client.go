// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package openssh

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"strconv"

	"go.jetpack.io/devbox/internal/debug"
)

type Client struct {
	Username       string
	Hostname       string
	ProjectDirName string
}

func (c *Client) Shell() error {
	cmd := c.cmd("-t")
	remoteCmd := fmt.Sprintf(`bash -l -c "start_devbox_shell.sh \"%s\""`, c.ProjectDirName)
	cmd.Args = append(cmd.Args, remoteCmd)
	debug.Log("running command: %s", cmd)

	// Setup stdin, stdout, stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *Client) Exec(remoteCmd string) ([]byte, error) {
	sshCmd := c.cmd()
	sshCmd.Args = append(sshCmd.Args, remoteCmd)
	debug.Log("cmd/exec: %s", sshCmd)
	return sshCmd.Output()
}

func (c *Client) cmd(sshArgs ...string) *exec.Cmd {
	host, port := c.hostPort()
	cmd := exec.Command("ssh", sshArgs...)
	cmd.Args = append(cmd.Args, destination(c.Username, host))

	// Add any necessary flags:
	if port != 0 && port != 22 {
		cmd.Args = append(cmd.Args, "-p", strconv.Itoa(port))
	}

	return cmd
}

func (c *Client) hostPort() (host string, port int) {
	host, portStr, err := net.SplitHostPort(c.Hostname)
	if err != nil {
		return c.Hostname, 22
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
