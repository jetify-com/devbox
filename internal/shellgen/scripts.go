package shellgen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/plugin"
)

const scriptsDir = ".devbox/gen/scripts"

// HooksFilename is the name of the file that contains the project's init-hooks and plugin hooks
const HooksFilename = ".hooks"

type devboxer interface {
	Config() *devconfig.Config
	Lockfile() *lock.File
	AllInstallablePackages() ([]*devpkg.Package, error)
	InstallablePackages() []*devpkg.Package
	IsUserShellFish() (bool, error)
	PluginManager() *plugin.Manager
	ProjectDir() string
}

// WriteScriptsToFiles writes scripts defined in devbox.json into files inside .devbox/gen/scripts.
// Scripts (and hooks) are persisted so that we can easily call them from devbox run (inside or outside shell).
func WriteScriptsToFiles(devbox devboxer) error {
	defer debug.FunctionTimer().End()
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
	pluginHooks, err := plugin.InitHooks(devbox.InstallablePackages(), devbox.ProjectDir())
	if err != nil {
		return errors.WithStack(err)
	}
	hooks := strings.Join(append(pluginHooks, devbox.Config().InitHook().String()), "\n\n")
	// always write it, even if there are no hooks, because scripts will source it.
	err = WriteScriptFile(devbox, HooksFilename, hooks)
	if err != nil {
		return errors.WithStack(err)
	}
	written[HooksFilename] = struct{}{}

	// Write scripts to files.
	for name, body := range devbox.Config().Scripts() {
		err = WriteScriptFile(devbox, name, ScriptBody(devbox, body.String()))
		if err != nil {
			return errors.WithStack(err)
		}
		written[name] = struct{}{}
	}

	// Delete any files that weren't written just now.
	for _, entry := range entries {
		scriptName := strings.TrimSuffix(entry.Name(), ".sh")
		if _, ok := written[scriptName]; !ok && !entry.IsDir() {
			err := os.Remove(ScriptPath(devbox.ProjectDir(), scriptName))
			if err != nil {
				debug.Log("failed to clean up script file %s, error = %s", entry.Name(), err) // no need to fail run
			}
		}
	}

	return nil
}

func WriteScriptFile(devbox devboxer, name string, body string) (err error) {
	script, err := os.Create(ScriptPath(devbox.ProjectDir(), name))
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
		// Fish cannot run scripts with `set -e`.
		// NOTE: Devbox scripts will run using `sh` for consistency. However,
		// init_hooks in a fish shell will run using `fish` shell, and need this
		// check.
		isFish, err := devbox.IsUserShellFish()
		if err != nil {
			return errors.WithStack(err)
		}
		if !isFish {
			body = fmt.Sprintf("set -e\n\n%s", body)
		}
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
