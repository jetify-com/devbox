package envsec

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/pkg/sandbox/runx"
)

var envCache map[string]string

func Env(projectDir string) (map[string]string, error) {

	defer debug.FunctionTimer().End()

	if envCache != nil {
		return envCache, nil
	}

	if err := ensureEnvsecInstalled(); err != nil {
		return nil, err
	}

	if err := ensureEnvsecInitialized(projectDir); err != nil {
		return nil, err
	}

	var err error
	envCache, err = envsecList(projectDir)

	return envCache, err
}

func ensureEnvsecInstalled() error {
	// In newer runx version this will return the paths
	paths, err := runx.Install("jetpack-io/envsec")
	if err != nil {
		return errors.Wrap(err, "failed to install envsec")
	}

	for _, path := range paths {
		os.Setenv("PATH", path+string(os.PathListSeparator)+os.Getenv("PATH"))
	}

	if !cmdutil.Exists("envsec") {
		return usererr.New("envsec is not installed or not in path")
	}
	return nil
}

func ensureEnvsecInitialized(projectDir string) error {
	cmd := exec.Command("envsec", "init", "--json-errors")
	cmd.Dir = projectDir
	var bufErr bytes.Buffer
	cmd.Stderr = &bufErr

	if err := cmd.Run(); err != nil {
		return handleError(&bufErr, err)
	}
	return nil
}

func envsecList(projectDir string) (map[string]string, error) {
	cmd := exec.Command(
		"envsec", "ls", "--show",
		"--format", "json",
		"--environment", "dev",
		"--json-errors")
	cmd.Dir = projectDir
	var bufErr bytes.Buffer
	cmd.Stderr = &bufErr
	out, err := cmd.Output()
	if err != nil {
		return nil, handleError(&bufErr, err)
	}
	var values []struct {
		Name  string `json:"name"`
		Value string `json:"value"`
	}

	if err := json.Unmarshal(out, &values); err != nil {
		return nil, errors.Wrapf(err, "failed to parse envsec output: %s", out)
	}

	m := map[string]string{}
	for _, v := range values {
		m[v.Name] = v.Value
	}
	return m, nil
}

func handleError(stderr *bytes.Buffer, err error) error {
	var errResponse struct {
		Error string `json:"error"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &errResponse); err == nil {
		return usererr.New(errResponse.Error)
	}
	return errors.WithStack(err)
}
