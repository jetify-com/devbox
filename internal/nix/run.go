package nix

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/debug"
)

func RunScript(projectDir string, cmdWithArgs string, env map[string]string) error {
	if cmdWithArgs == "" {
		return errors.New("attempted to run an empty command or script")
	}

	envPairs := []string{}
	for k, v := range env {
		envPairs = append(envPairs, fmt.Sprintf("%s=%s", k, v))
	}

	// Try to find sh in the PATH, if not, default to a well known absolute path.
	shPath, err := exec.LookPath("sh")
	if err != nil {
		shPath = "/bin/sh"
	}
	cmd := exec.Command(shPath, "-c", cmdWithArgs)
	cmd.Env = envPairs
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
