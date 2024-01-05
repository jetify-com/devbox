package envsec

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/envsec/pkg/envsec"
	"go.jetpack.io/pkg/envvar"
)

var (
	envCache     map[string]string
	binPathCache string
)

func Env(ctx context.Context, projectDir, environment string) (map[string]string, error) {
	defer debug.FunctionTimer().End()

	if envCache != nil {
		return envCache, nil
	}

	if err := ensureInitialized(projectDir); err != nil {
		return nil, err
	}

	envCache, err := envsecList(ctx, projectDir, environment)

	return envCache, err
}

func EnsureInstalled(ctx context.Context) (string, error) {
	if binPathCache != "" {
		return binPathCache, nil
	}

	if path, err := exec.LookPath("envsec"); err == nil {
		binPathCache = path
		return binPathCache, nil
	}

	paths, err := pkgtype.RunXClient().Install(ctx, "jetpack-io/envsec@v0.0.15")
	if err != nil {
		return "", errors.Wrap(err, "failed to install envsec")
	}

	if len(paths) == 0 {
		return "", usererr.New("envsec is not installed or not in path")
	}

	binPathCache = filepath.Join(paths[0], "envsec")
	return binPathCache, nil
}

func ensureInitialized(projectDir string) error {
	envsec := DefaultEnvsec(os.Stderr, projectDir)
	_, err := envsec.ProjectConfig()
	if err != nil {
		return errors.New(
			"envsec project is not initialized. Use `devbox envsec init` to initialize",
		)
	}
	return nil
}

func envsecList(
	ctx context.Context,
	projectDir, environment string,
) (map[string]string, error) {
	binPath, err := EnsureInstalled(ctx)
	if err != nil {
		return nil, err
	}
	cmd := exec.Command(
		binPath, "ls", "--show",
		"--format", "json",
		"--environment", environment,
		"--json-errors")
	cmd.Dir = projectDir
	if build.IsDev {
		// Ensure that devbox and envsec build envs are the same
		cmd.Env = append(os.Environ(), "ENVSEC_BUILD_ENV=dev")
	}

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

func DefaultEnvsec(stderr io.Writer, workingDir string) *envsec.Envsec {
	return &envsec.Envsec{
		APIHost: build.JetpackAPIHost(),
		Auth: envsec.AuthConfig{
			ClientID: envvar.Get("ENVSEC_CLIENT_ID", build.ClientID()),
			Issuer:   envvar.Get("ENVSEC_ISSUER", build.Issuer()),
		},
		IsDev:      build.IsDev,
		Stderr:     stderr,
		WorkingDir: workingDir,
	}
}
