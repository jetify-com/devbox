package shellgen

import (
	"bytes"
	_ "embed"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"text/template"

	"github.com/pkg/errors"
	"go.jetify.com/devbox/internal/boxcli/featureflag"
	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/plugin"
)

//go:embed tmpl/script-wrapper.tmpl
var scriptWrapperTmplString string
var scriptWrapperTmpl = template.Must(template.New("script-wrapper").Parse(scriptWrapperTmplString))

const scriptsDir = ".devbox/gen/scripts"

const HooksFilename = ".hooks"

// ArbitraryCmdFilename is the script `devbox run` uses to run a command that
// isn't a devbox.json script. Every invocation writes the same contents, so it
// is shared, and WriteScriptsToFiles must not clean it up.
const ArbitraryCmdFilename = ".cmd"

type devboxer interface {
	Config() *devconfig.Config
	Lockfile() *lock.File
	InstallablePackages() []*devpkg.Package
	PluginManager() *plugin.Manager
	ProjectDir() string
	SkipInitHookEnvName() string
}

// WriteScriptsToFiles writes scripts defined in devbox.json into files inside .devbox/gen/scripts.
// Scripts (and hooks) are persisted so that we can easily call them from devbox run (inside or outside shell).
func WriteScriptsToFiles(devbox devboxer) error {
	defer debug.FunctionTimer().End()
	err := os.MkdirAll(filepath.Join(devbox.ProjectDir(), scriptsDir), 0o755) // Ensure directory exists.
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
	// always write it, even if there are no hooks, because scripts will source it.
	err = writeRawInitHookFile(devbox, devbox.Config().InitHook().String())
	if err != nil {
		return errors.WithStack(err)
	}
	written[HooksFilename] = struct{}{}
	// Another `devbox run` in this project may be about to execute it.
	written[ArbitraryCmdFilename] = struct{}{}

	// Write scripts to files.
	for name, body := range devbox.Config().Scripts() {
		scriptBody, err := ScriptBody(devbox, body.String())
		if err != nil {
			return errors.WithStack(err)
		}
		err = WriteScriptFile(devbox, name, scriptBody)
		if err != nil {
			return errors.WithStack(err)
		}
		written[name] = struct{}{}
	}

	// Delete any files that weren't written just now.
	// Only look at .sh files, so that we skip the temp files of concurrent
	// writers.
	for _, entry := range entries {
		scriptName, isScript := strings.CutSuffix(entry.Name(), ".sh")
		if _, ok := written[scriptName]; !ok && isScript && !entry.IsDir() {
			err := os.Remove(ScriptPath(devbox.ProjectDir(), scriptName))
			if err != nil {
				slog.Debug("failed to clean up script file", "file", entry.Name(), "err", err) // no need to fail run
			}
		}
	}

	return nil
}

func writeRawInitHookFile(devbox devboxer, body string) error {
	return writeScriptFileAtomic(ScriptPath(devbox.ProjectDir(), HooksFilename), body)
}

func WriteScriptFile(devbox devboxer, name, body string) error {
	if featureflag.ScriptExitOnError.Enabled() {
		// NOTE: Devbox scripts run using `sh` for consistency.
		body = fmt.Sprintf("set -e\n\n%s", body)
	}
	return writeScriptFileAtomic(ScriptPath(devbox.ProjectDir(), name), body)
}

// writeScriptFileAtomic writes an executable script to path atomically.
// Concurrent `devbox run` invocations in the same project rewrite the same
// scripts, so writing in place could let one of them execute an empty,
// partial, or not yet executable file.
func writeScriptFileAtomic(path, body string) error {
	tmp, err := writeTempFile(filepath.Dir(path), "."+filepath.Base(path)+"-*", body)
	if tmp != "" {
		defer os.Remove(tmp) // no-op after a successful rename
	}
	if err != nil {
		return err
	}
	if err := os.Chmod(tmp, 0o755); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(os.Rename(tmp, path))
}

// writeTempFile creates a temp file in dir, writes body to it, and closes it.
// It returns the file's name whenever the file was created, even on error.
//
// It holds syscall.ForkLock for reading while the file is open, so that no
// other goroutine forks in that window. A forked child inherits the writable
// fd until it execs, and on Linux exec'ing the script while any process holds
// it open for writing fails with ETXTBSY ("Text file busy").
// See https://go.dev/issue/22315.
func writeTempFile(dir, pattern, body string) (string, error) {
	syscall.ForkLock.RLock()
	defer syscall.ForkLock.RUnlock()

	tmp, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if _, err := tmp.WriteString(body); err != nil {
		tmp.Close()
		return tmp.Name(), errors.WithStack(err)
	}
	return tmp.Name(), errors.WithStack(tmp.Close())
}

func ScriptPath(projectDir, scriptName string) string {
	return filepath.Join(projectDir, scriptsDir, scriptName+".sh")
}

func ScriptBody(d devboxer, body string) (string, error) {
	var buf bytes.Buffer
	err := scriptWrapperTmpl.Execute(&buf, map[string]string{
		"Body":             body,
		"SkipInitHookHash": d.SkipInitHookEnvName(),
		"InitHookPath":     ScriptPath(d.ProjectDir(), HooksFilename),
	})
	if err != nil {
		return "", errors.WithStack(err)
	}
	return buf.String(), nil
}
