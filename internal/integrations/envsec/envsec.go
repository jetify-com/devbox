package envsec

import (
	"encoding/json"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cmdutil"
)

func Env(projectDir string) (map[string]string, error) {

	if err := ensureEnvsecInstalled(); err != nil {
		return nil, err
	}

	if err := ensureEnvsecInitialized(); err != nil {
		return nil, err
	}

	return envsecList(projectDir)
}

func ensureEnvsecInstalled() error {
	if !cmdutil.Exists("envsec") {
		return usererr.New("envsec is not installed or not in path")
	}
	return nil
}

func ensureEnvsecInitialized() error {
	cmd := exec.Command("envsec", "init")
	// TODO handle user not logged in
	// envsec init is currently broken in that it exits with 0 even if the user is not logged in
	return cmd.Run()
}

func envsecList(projectDir string) (map[string]string, error) {
	cmd := exec.Command(
		"envsec", "ls", "--show",
		"--format", "json",
		"--environment", "dev")
	cmd.Dir = projectDir
	out, err := cmd.Output()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var values []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal(out, &values); err != nil {
		return nil, errors.Wrap(err, "failed to parse envsec output")
	}

	m := map[string]string{}
	for _, v := range values {
		m[v.Name] = v.Value
	}
	return m, nil
}
