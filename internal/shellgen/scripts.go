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
	pluginHooks, err := devbox.PluginManager().InitHooks(
		devbox.InstallablePackages(),
		devbox.Config().Include,
	)
	if err != nil {
		return errors.WithStack(err)
	}
	hooks := strings.Join(append(pluginHooks, devbox.Config().InitHook().String()), "\n\n")
	// always write it, even if there are no hooks, because scripts will source it.
	err = writeHookFile(devbox, hooks)
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

func writeHookFile(devbox devboxer, body string) (err error) {
	script, err := createScriptFile(devbox, HooksFilename)
	if err != nil {
		return errors.WithStack(err)
	}
	defer script.Close() // best effort: close file

	_, err = script.WriteString(body)
	return errors.WithStack(err)
}

func WriteScriptFile(devbox devboxer, name string, body string) (err error) {
	script, err := createScriptFile(devbox, name)
	if err != nil {
		return errors.WithStack(err)
	}
	defer script.Close() // best effort: close file

	if featureflag.ScriptExitOnError.Enabled() {
		// NOTE: Devbox scripts will run using `sh` for consistency.
		// However, we need to disable this for `fish` shell if/when we allow this for init_hooks,
		// since init_hooks run in the host shell, and not `sh`.
		body = fmt.Sprintf("set -e\n\n%s", body)
	}
	_, err = script.WriteString(body)
	return errors.WithStack(err)
}

func createScriptFile(devbox devboxer, name string) (script *os.File, err error) {
	script, err = os.Create(ScriptPath(devbox.ProjectDir(), name))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer func() {
		// best effort: close file if there was some subsequent error
		if err != nil {
			_ = script.Close()
		}
	}()

	err = script.Chmod(0755)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return script, nil
}

func ScriptPath(projectDir, scriptName string) string {
	return filepath.Join(projectDir, scriptsDir, scriptName+".sh")
}

func ScriptBody(d devboxer, body string) string {
	return fmt.Sprintf(". %s\n\n%s", ScriptPath(d.ProjectDir(), HooksFilename), body)
}
