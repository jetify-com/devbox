package git

import (
	"bytes"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"
)

const rejectedErrorText = "rejected"

var ErrRejected = errors.New(rejectedErrorText)
var ErrNotAGitRepo = errors.New(rejectedErrorText)

func Push(dir string, force bool) error {

	if hasUncommitedChanges(dir) {
		if err := createCommit(dir); err != nil {
			return err
		}
	}

	cmd := exec.Command("git", "push")
	if force {
		cmd.Args = append(cmd.Args, "--force")
	}
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = io.MultiWriter(os.Stderr, buf)
	cmd.Stdout = os.Stdout
	cmd.Dir = dir
	err := cmd.Run()
	if strings.Contains(buf.String(), rejectedErrorText) {
		return ErrRejected
	}
	return errors.WithStack(err)
}

func hasUncommitedChanges(dir string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return false
	}
	return len(out) > 0
}

func createCommit(dir string) error {
	cmd := exec.Command("git", "add", ".")
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		return err
	}
	cmd = exec.Command("git", "commit", "-m", "devbox commit")
	cmd.Dir = dir
	return cmd.Run()
}
