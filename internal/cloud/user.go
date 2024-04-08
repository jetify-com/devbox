// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"bytes"
	"os/exec"
	"regexp"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
)

var githubSSHRegexp = regexp.MustCompile("Hi (.+)! You've successfully authenticated, " +
	"but GitHub does not provide shell access")

// queryGithubUsername attempts to make an ssh connection to github, which replies
// with a friendly rejection message that contains the user's username (if they have github
// credentials set up correctly). We parse the username from the error message.
func queryGithubUsername() (string, error) {
	cmd := exec.Command("ssh", "-T", "-o", "NumberOfPasswordPrompts=0", "git@github.com")
	var bufOut, bufErr bytes.Buffer
	cmd.Stdin = nil
	cmd.Stdout = &bufOut
	cmd.Stderr = &bufErr
	err := cmd.Run()
	if err != nil {
		if e := (&exec.ExitError{}); errors.As(err, &e) && e.ExitCode() == 1 {
			// This is the Happy case, and we can parse out the error message
			debug.Log(
				"Received expected (this is good) error for cmd `%s` had exit code 1 with stderr: %v", cmd,
				bufErr.String(),
			)
			return parseUsernameFromErrorMessage(bufErr.String()), nil
		}
		// This is the sad case, and we should let the caller figure out how to proceed with the user
		debug.Log("error from command `%s`: %v, out: %v, stderr: %v", cmd, err, bufOut.String(), bufErr.String())
		return "", errors.WithStack(err)
	}

	return "", nil
}

func parseUsernameFromErrorMessage(errorMessage string) string {
	matchedUsernames := githubSSHRegexp.FindSubmatch([]byte(errorMessage))
	if len(matchedUsernames) < 2 {
		debug.Log("Did not find a username from github. Message is: %s", errorMessage)
		return ""
	}
	debug.Log("matched username from github is: %s\n", matchedUsernames[1])
	return string(matchedUsernames[1])
}
