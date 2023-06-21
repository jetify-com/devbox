package filegen

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
)

const scriptsDir = ".devbox/gen/scripts"

// HooksFilename is the name of the file that contains the project's init-hooks and plugin hooks
const HooksFilename = ".hooks"

type devboxer interface {
	Config() *devconfig.Config
	FlakePlan(context.Context) (*plansdk.FlakePlan, error)
	PackagesAsInputs() []*nix.Input
	ProjectDir() string
}

// WriteScriptsToFiles writes scripts defined in devbox.json into files inside .devbox/gen/scripts.
// Scripts (and hooks) are persisted so that we can easily call them from devbox run (inside or outside shell).
func WriteScriptsToFiles(devbox devboxer) error {
	err := os.MkdirAll(filepath.Join(devbox.ProjectDir(), scriptsDir), 0755) // Ensure directory exists.
	if err != nil {
		return errors.WithStack(err)
	}

	// Read dir contents before writing, so we can clean up later.
	entries, err := os.ReadDir(filepath.Join(devbox.ProjectDir(), scriptsDir))
	if err != nil {
		return errors.WithStack(err)
	}

	// Write all hooks to a file.
	written := map[string]struct{}{} // set semantics; value is irrelevant
	pluginHooks, err := plugin.InitHooks(devbox.PackagesAsInputs(), devbox.ProjectDir())
	if err != nil {
		return errors.WithStack(err)
	}
	hooks := strings.Join(append(pluginHooks, devbox.Config().InitHook().String()), "\n\n")
	// always write it, even if there are no hooks, because scripts will source it.
	err = WriteScriptFile(devbox, HooksFilename, hooks)
	if err != nil {
		return errors.WithStack(err)
	}
	written[ScriptPath(devbox.ProjectDir(), HooksFilename)] = struct{}{}

	// Write scripts to files.
	for name, body := range devbox.Config().Scripts() {
		err = WriteScriptFile(devbox, name, ScriptBody(devbox, body.String()))
		if err != nil {
			return errors.WithStack(err)
		}
		written[ScriptPath(devbox.ProjectDir(), name)] = struct{}{}
	}

	// Delete any files that weren't written just now.
	for _, entry := range entries {
		if _, ok := written[entry.Name()]; !ok && !entry.IsDir() {
			err := os.Remove(ScriptPath(devbox.ProjectDir(), entry.Name()))
			if err != nil {
				debug.Log("failed to clean up script file %s, error = %s", entry.Name(), err) // no need to fail run
			}
		}
	}

	return nil
}

func WriteScriptFile(d devboxer, name string, body string) (err error) {
	script, err := os.Create(ScriptPath(d.ProjectDir(), name))
	if err != nil {
		return errors.WithStack(err)
	}
	defer func() {
		cerr := script.Close()
		if err == nil {
			err = cerr
		}
	}()
	err = script.Chmod(0755)
	if err != nil {
		return errors.WithStack(err)
	}

	if featureflag.ScriptExitOnError.Enabled() {
		body = fmt.Sprintf("set -e\n\n%s", body)
	}
	_, err = script.WriteString(body)
	return errors.WithStack(err)
}

func ScriptPath(projectDir, scriptName string) string {
	return filepath.Join(projectDir, scriptsDir, scriptName+".sh")
}

func ScriptBody(d devboxer, body string) string {
	return fmt.Sprintf(". %s\n\n%s", ScriptPath(d.ProjectDir(), HooksFilename), body)
}
