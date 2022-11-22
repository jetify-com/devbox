package sshclient

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

type Client struct {
	Username string
	Hostname string
	Port     int
}

func (c *Client) Shell() error {
	cmd := c.cmd()

	// Setup stdin, stdout, stderr
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func (c *Client) Exec(remoteCmd string) ([]byte, error) {
	sshCmd := c.cmd()

	sshCmd.Args = append(sshCmd.Args, remoteCmd)
	bytes, err := sshCmd.Output()
	var exerr *exec.ExitError
	if errors.As(err, &exerr) {
		// Ignore exit code errors and just return the output
		return bytes, nil
	}
	return bytes, err
}

func (c *Client) cmd() *exec.Cmd {
	cmd := exec.Command("ssh", destination(c.Username, c.Hostname))

	// Add any necessary flags:
	if c.Port != 0 {
		cmd.Args = append(cmd.Args, "-p", strconv.Itoa(c.Port))
	}

	return cmd
}

func destination(username, hostname string) string {
	result := hostname
	if username != "" {
		result = username + "@" + result
	}

	return result
}
