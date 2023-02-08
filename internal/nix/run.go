package nix

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
)

func RunScript(
	nixShellFilePath, nixFlakesFilePath string,
	projectDir string,
	cmdWithArgs string,
	additionalEnv []string,
) error {
	if cmdWithArgs == "" {
		return errors.New("attempted to run an empty command or script")
	}

	vaf, err := PrintDevEnv(nixShellFilePath, nixFlakesFilePath)
	if err != nil {
		return err
	}

	nixEnv := []string{}
	for k, v := range vaf.Variables {
		if v.Type == "exported" {
			nixEnv = append(nixEnv, fmt.Sprintf("%s=%s", k, v.Value.(string)))
		}
	}

	// Overwrite/leak whitelisted vars into nixEnv:
	for name, leak := range leakVarsForRun {
		if leak {
			nixEnv = append(nixEnv, fmt.Sprintf("%s=%s", name, os.Getenv(name)))
		}
	}

	// Try to find sh in the PATH, if not, default to a well known absolute path.
	shPath, err := exec.LookPath("sh")
	if err != nil {
		shPath = "/bin/sh"
	}
	cmd := exec.Command(shPath, "-c", cmdWithArgs)
	cmd.Env = append(nixEnv, additionalEnv...)
	cmd.Dir = projectDir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	debug.Log("Executing: %v", cmd.Args)
	err = cmd.Run()
	if err != nil {
		// Report error as exec error when executing scripts.
		err = usererr.NewExecError(err)
	}
	return errors.WithStack(err)
}

// leakVarsForRun contains a list of variables that, if set in the host, will be copied
// to the environment of devbox run. If they're NOT set in the host, they will be set
// to an empty value for devbox run. NOTE: we want to keep this list AS SMALL AS POSSIBLE.
// The longer this list, the less "pure" devbox run becomes.
//
// In particular, this list should be much smaller than that of devbox shell, since we
// do want to allow more parts of the host environment to leak into a shell session, so
// that the shell session is easy to use for our users. However, in devbox run, we value
// reproducibility above interactive ease-of-use.
var leakVarsForRun = map[string]bool{
	"HOME": true, // Without this, HOME is set to /homeless-shelter and most programs fail.

	// Where to write temporary files. nix print-dev-env sets these to an unwriteable path,
	// so we override that here with whatever the host has set.
	"TMP":     true,
	"TEMP":    true,
	"TMPDIR":  true,
	"TEMPDIR": true,
}
