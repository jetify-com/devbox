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
	cmd := exec.Command("ssh", "-T", "git@github.com")
	var bufOut, bufErr bytes.Buffer
	cmd.Stdout = &bufOut
	cmd.Stderr = &bufErr
	err := cmd.Run()

	var errorMessage string
	if err != nil {
		if e := (&exec.ExitError{}); errors.As(err, &e) && e.ExitCode() == 1 {
			debug.Log("Received expected (this is good) error for cmd `%s` had exit code 1 with stderr: %v", cmd,
				bufErr.String())
			errorMessage = bufErr.String()
		} else {
			debug.Log("error from command `%s`: %v, out: %v, stderr: %v", cmd, err, bufOut.String(), bufErr.String())
			return "", errors.WithStack(err)
		}
	}

	// parse output
	matchedUsernames := githubSSHRegexp.FindSubmatch([]byte(errorMessage))
	if len(matchedUsernames) < 2 {
		debug.Log("Did not find a username from github. Message is: %s", errorMessage)
		return "", nil
	}
	debug.Log("matched username from github is: %s\n", matchedUsernames[1])
	return string(matchedUsernames[1]), nil
}
