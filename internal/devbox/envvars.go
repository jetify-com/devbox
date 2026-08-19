// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"slices"
	"strings"

	"go.jetify.com/devbox/internal/devbox/envpath"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/ux"
)

const devboxSetPrefix = "__DEVBOX_SET_"

// envNameRegexp matches a valid POSIX shell environment variable name: it must
// start with a letter or underscore and contain only letters, digits, and
// underscores. Names that don't match (e.g. a "//" comment key in devbox.json's
// env block) can't be exported without producing invalid shell syntax.
var envNameRegexp = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

func isValidEnvName(name string) bool {
	return envNameRegexp.MatchString(name)
}

// warnInvalidEnvNames prints a single warning naming any environment variables
// that were skipped because they aren't valid shell identifiers. The most common
// cause is a "//" comment key in a devbox.json env block, which devbox would
// otherwise emit as `export //=...` and break the entire shell with a cryptic
// error.
func warnInvalidEnvNames(w io.Writer, names []string) {
	if len(names) == 0 {
		return
	}
	quoted := make([]string, len(names))
	for i, name := range names {
		quoted[i] = fmt.Sprintf("%q", name)
	}
	ux.Fwarningf(
		w,
		"Skipping %d environment variable(s) with invalid names: %s.\n"+
			"Environment variable names must match ^[a-zA-Z_][a-zA-Z0-9_]*$. "+
			"If these are \"//\" comments in your devbox.json env block, remove or rename them.\n",
		len(names),
		strings.Join(quoted, ", "),
	)
}

// exportify formats vars as a line-separated string of shell export statements.
// Each line is of the form `export key="value";` with any special characters in
// value escaped. This means that the shell will always interpret values as
// literal strings; no variable expansion or command substitution will take
// place.
func exportify(w io.Writer, vars map[string]string) string {
	keys := make([]string, len(vars))
	i := 0
	for k := range vars {
		keys[i] = k
		i++
	}
	slices.Sort(keys) // for reproducibility

	var invalidNames []string
	strb := strings.Builder{}
	for _, key := range keys {
		if strings.HasPrefix(key, "BASH_FUNC_") && strings.HasSuffix(key, "%%") {
			// Bash function
			funcName := strings.TrimSuffix(key, "%%")
			funcName = strings.TrimPrefix(funcName, "BASH_FUNC_")
			strb.WriteString(funcName)
			strb.WriteString(" ")
			strb.WriteString(vars[key])
			strb.WriteString("\nexport -f ")
			strb.WriteString(funcName)
			strb.WriteString("\n")
		} else {
			// Regular variable. Skip names that aren't valid shell
			// identifiers; exporting them would produce invalid syntax that
			// breaks the whole shell (e.g. `export //=...`).
			if !isValidEnvName(key) {
				invalidNames = append(invalidNames, key)
				continue
			}
			strb.WriteString("export ")
			strb.WriteString(key)
			strb.WriteString(`="`)
			for _, r := range vars[key] {
				switch r {
				// Special characters inside double quotes:
				// https://pubs.opengroup.org/onlinepubs/009604499/utilities/xcu_chap02.html#tag_02_02_03
				case '$', '`', '"', '\\', '\n':
					strb.WriteRune('\\')
				}
				strb.WriteRune(r)
			}
			strb.WriteString("\";\n")
		}
	}
	warnInvalidEnvNames(w, invalidNames)
	return strings.TrimSpace(strb.String())
}

// exportifyNushell formats vars as nushell environment variable assignments.
// Each line is of the form `$env.KEY = "value"` with special characters escaped.
func exportifyNushell(w io.Writer, vars map[string]string) string {
	// Nushell protected environment variables that cannot be set manually
	// See: https://www.nushell.sh/book/environment.html#automatic-environment-variables
	protectedVars := map[string]bool{
		"CURRENT_FILE":    true,
		"FILE_PWD":        true,
		"LAST_EXIT_CODE":  true,
		"CMD_DURATION_MS": true,
		"NU_VERSION":      true,
		"PWD":             true, // Nushell manages this automatically
	}

	keys := make([]string, len(vars))
	i := 0
	for k := range vars {
		keys[i] = k
		i++
	}
	slices.Sort(keys) // for reproducibility

	var invalidNames []string
	strb := strings.Builder{}
	for _, key := range keys {
		// Skip bash functions for nushell
		if strings.HasPrefix(key, "BASH_FUNC_") && strings.HasSuffix(key, "%%") {
			continue
		}

		// Skip nushell protected environment variables
		if protectedVars[key] {
			continue
		}

		// Skip names that aren't valid environment variable identifiers.
		if !isValidEnvName(key) {
			invalidNames = append(invalidNames, key)
			continue
		}

		// Nushell environment variable syntax: $env.KEY = "value"
		strb.WriteString("$env.")
		strb.WriteString(key)
		strb.WriteString(` = "`)
		for _, r := range vars[key] {
			switch r {
			// Escape special characters for nushell double-quoted strings
			case '"', '\\':
				strb.WriteRune('\\')
			}
			strb.WriteRune(r)
		}
		strb.WriteString("\"\n")
	}
	warnInvalidEnvNames(w, invalidNames)
	return strings.TrimSpace(strb.String())
}

// onlyModifiedEnvVars returns the subset of env whose values are new or differ
// from the ambient environment. Variables whose value already matches the
// ambient environment are omitted: re-exporting them is redundant, and at worst
// it breaks `eval "$(devbox shellenv)"` when the user's shell marks some of
// those variables read-only (e.g. PROFILEREAD on openSUSE, which produces
// "read-only variable: PROFILEREAD"). See issue #2826.
func onlyModifiedEnvVars(env, ambient map[string]string) map[string]string {
	modified := make(map[string]string, len(env))
	for key, val := range env {
		if ambientVal, ok := ambient[key]; !ok || ambientVal != val {
			modified[key] = val
		}
	}
	return modified
}

// addEnvIfNotPreviouslySetByDevbox adds the key-value pairs from new to existing,
// but only if the key was not previously set by devbox
// Caveat, this won't mark the values as set by devbox automatically. Instead,
// you need to call markEnvAsSetByDevbox when you are done setting variables.
// This is so you can add variables from multiple sources (e.g. plugin, devbox.json)
// that may build on each other (e.g. PATH=$PATH:...)
func addEnvIfNotPreviouslySetByDevbox(existing, new map[string]string) {
	for k, v := range new {
		if _, alreadySet := existing[devboxSetPrefix+k]; !alreadySet {
			existing[k] = v
		}
	}
}

func markEnvsAsSetByDevbox(envs ...map[string]string) {
	for _, env := range envs {
		for key := range env {
			env[devboxSetPrefix+key] = "1"
		}
	}
}

// IsEnvEnabled checks if the devbox environment is enabled.
// This allows us to differentiate between global and
// individual project shells.
func (d *Devbox) IsEnvEnabled() bool {
	fakeEnv := map[string]string{}
	// the Stack is initialized in the fakeEnv, from the state in the real os.Environ
	pathStack := envpath.Stack(fakeEnv, envir.PairsToMap(os.Environ()))
	return pathStack.Has(d.ProjectDirHash())
}

func (d *Devbox) SkipInitHookEnvName() string {
	return "__DEVBOX_SKIP_INIT_HOOK_" + d.ProjectDirHash()
}
