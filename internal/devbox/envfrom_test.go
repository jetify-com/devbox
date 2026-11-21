// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/devbox/envpath"
	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/nix"
)

// writeMockCommand writes a "secrets" script to the project's profile.
func writeMockCommand(t *testing.T, box *Devbox, script string) {
	t.Helper()
	dir := nix.ProfileBinPath(box.projectDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "secrets"), []byte("#!/bin/sh\n"+script), 0o755))
}

func envFromDevboxForTesting(t *testing.T, envFrom ...string) (*Devbox, *bytes.Buffer) {
	t.Helper()
	dir := t.TempDir()
	stderr := &bytes.Buffer{}
	return &Devbox{
		cfg: &devconfig.Config{Root: configfile.ConfigFile{
			EnvFrom:     envFrom,
			AbsRootPath: filepath.Join(dir, "devbox.json"),
		}},
		projectDir: dir,
		stderr:     stderr,
	}, stderr
}

func profileEnv(box *Devbox) map[string]string {
	return map[string]string{"PATH": nix.ProfileBinPath(box.projectDir)}
}

func writeProjectFile(t *testing.T, box *Devbox, name, content string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(box.projectDir, name), []byte(content), 0o644))
}

func TestParseEnvJSON(t *testing.T) {
	got, nested, err := parseEnvJSON([]byte(
		`{"DB_PASSWORD":"pa$$word","PORT":5432,"VERSION":"1.10","DEBUG":true,"EMPTY":null}`,
	))
	require.NoError(t, err)
	assert.Empty(t, nested)
	assert.Equal(t, map[string]string{
		"DB_PASSWORD": "pa$$word",
		"PORT":        "5432",
		"VERSION":     "1.10",
		"DEBUG":       "true",
		"EMPTY":       "",
	}, got)
}

func TestParseEnvJSON_SkipsNestedValues(t *testing.T) {
	got, nested, err := parseEnvJSON([]byte(`{"DB":{"HOST":"localhost"},"LIST":["a"],"PASS":"x"}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PASS": "x"}, got)
	assert.ElementsMatch(t, []string{"DB", "LIST"}, nested)
}

func TestParseEnvJSON_Invalid(t *testing.T) {
	for _, out := range []string{`["a"]`, `{"A":"x"} {"B":"y"}`, `A=x`, ``} {
		_, _, err := parseEnvJSON([]byte(out))
		assert.Error(t, err, out)
	}
}

func TestExecEnv_WarnsOnNestedValues(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo '{\"DB\":{\"HOST\":\"h\"},\"PASS\":\"x\"}'\n")

	env := box.execEnv(t.Context(), "secrets", profileEnv(box))
	assert.Equal(t, map[string]string{"PASS": "x"}, env)
	assert.Contains(t, stderr.String(), `Ignoring nested values from env_from "exec:secrets" for keys: DB.`)
}

func TestExecProcessEnv(t *testing.T) {
	box, _ := envFromDevboxForTesting(t)
	box.cfg.Root.Env = map[string]string{"SOPS_AGE_KEY_FILE": "$PWD/key.txt", "FROM_FLAG": "config"}
	box.env = map[string]string{"FROM_FLAG": "flag"}

	box.cfg.Root.Env["PATH"] = "/repo/tools:$PATH"
	env := box.execProcessEnv(map[string]string{"PATH": "/profile/bin", "HOME": "/home/me"}, nil)
	assert.Equal(t, filepath.Join(box.projectDir, "key.txt"), env["SOPS_AGE_KEY_FILE"])
	assert.Equal(t, "/home/me", env["HOME"])
	assert.Equal(t, "flag", env["FROM_FLAG"])
	profile, runx := nix.ProfileBinPath(box.projectDir), runxBinPath(box.projectDir)
	t.Setenv("PATH", "/host/bin")
	env = box.execProcessEnv(map[string]string{"PATH": "/nix/env/bin"}, nil)
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/nix/env/bin", "/host/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])

	// With OmitNixEnv, the computed PATH already contains the host PATH.
	env = box.execProcessEnv(map[string]string{"PATH": "/host/bin"}, nil)
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/host/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])

	env = box.execProcessEnv(map[string]string{"PATH": "/nix/env/bin", "DEVBOX_PURE_SHELL": "1"}, nil)
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/nix/env/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])
}

func TestExecEnv_RunsInProcessEnv(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box,
		"echo 'touch your key' >&2\n"+
			"echo \"{\\\"KEY_FILE\\\":\\\"$SOPS_AGE_KEY_FILE\\\",\\\"DIR\\\":\\\"$(pwd -P)\\\",\\\"ARG\\\":\\\"$1\\\"}\"\n")
	procEnv := profileEnv(box)
	procEnv["PATH"] = envpath.JoinPathLists(procEnv["PATH"], os.Getenv("PATH"))
	procEnv["SOPS_AGE_KEY_FILE"] = "/keys.txt"

	env := box.execEnv(t.Context(), "secrets 'a b' | cat", procEnv)
	dir, err := filepath.EvalSymlinks(box.projectDir)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"KEY_FILE": "/keys.txt", "DIR": dir, "ARG": "a b"}, env)
	assert.Contains(t, stderr.String(), "touch your key")
}

func TestExecEnv_RunsOnce(t *testing.T) {
	box, _ := envFromDevboxForTesting(t)
	counter := filepath.Join(box.projectDir, "calls")
	writeMockCommand(t, box, "echo call >> '"+counter+"'\necho '{\"API_KEY\":\"abc\"}'\n")

	for range 2 {
		env := box.execEnv(t.Context(), "secrets", profileEnv(box))
		assert.Equal(t, map[string]string{"API_KEY": "abc"}, env)
	}
	calls, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(calls), "call"))
}

func TestExecEnv_WarnsOnceOnFailure(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo 'no key' >&2\nexit 1\n")

	for range 2 {
		assert.Empty(t, box.execEnv(t.Context(), "secrets", profileEnv(box)))
	}
	assert.Equal(t, 1, strings.Count(stderr.String(), "no key"))
	assert.Equal(t, 1, strings.Count(stderr.String(), `Ignoring env_from "exec:secrets": command failed`))
}

func TestExecEnv_WarnsOnInvalidOutput(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo 'PASS=x'\n")

	assert.Empty(t, box.execEnv(t.Context(), "secrets", profileEnv(box)))
	assert.Contains(t, stderr.String(), `Ignoring env_from "exec:secrets": output is not a flat JSON object`)
}

func TestExecEnv_DoesNotWaitForBackgroundProcesses(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo '{\"PASS\":\"x\"}'\nsleep 5 &\n")

	procEnv := profileEnv(box)
	procEnv["PATH"] = envpath.JoinPathLists(procEnv["PATH"], os.Getenv("PATH"))
	start := time.Now()
	env := box.execEnv(t.Context(), "secrets", procEnv)
	assert.Less(t, time.Since(start), 4*time.Second)
	assert.Equal(t, map[string]string{"PASS": "x"}, env)
	assert.Empty(t, stderr.String())
}

func TestExecEnv_IgnoresPath(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo '{\"PATH\":\"/evil\",\"PASS\":\"x\"}'\n")

	env := box.execEnv(t.Context(), "secrets", profileEnv(box))
	assert.Equal(t, map[string]string{"PASS": "x"}, env)
	assert.Contains(t, stderr.String(), "Ignoring PATH")
}

func TestCollectSecretEnvKeys(t *testing.T) {
	got := collectSecretEnvKeys([]string{"PASS", "TOKEN", "KEY"}, map[string]string{
		"DSN":     "postgres://u:$PASS@h",
		"BRACED":  "${PASS}",
		"PLAIN":   "value",
		"PASSING": "$PASSWORD",
		"TOKEN":   "from_config",
		"KEY":     "${KEY}-suffix",
	})
	assert.ElementsMatch(t, []string{"PASS", "DSN", "BRACED", "KEY"}, got)
}

func TestConfigEnvs_Exec(t *testing.T) {
	box, _ := envFromDevboxForTesting(t, "exec:secrets")
	box.cfg.Root.Env = map[string]string{
		"OVERRIDE": "from_config",
		"HOME_DIR": "$HOME",
		"DSN":      "u:$PASS@h",
	}
	writeMockCommand(t, box, "echo '{\"PASS\":\"pa$$word\",\"OVERRIDE\":\"from_exec\"}'\n")
	existing := profileEnv(box)
	existing["HOME"] = "/home/me"
	existing["PASS"] = "from_host"

	env, err := box.configEnvs(t.Context(), existing)
	require.NoError(t, err)
	assert.Equal(t, "pa$$word", env["PASS"])
	assert.Equal(t, "u:pa$$word@h", env["DSN"])
	assert.Equal(t, "from_config", env["OVERRIDE"])
	assert.Equal(t, "/home/me", env["HOME_DIR"])
	assert.ElementsMatch(t, []string{"PASS", "DSN"}, box.secretEnvKeys)
}

func TestConfigEnvs_MergesSourcesInOrder(t *testing.T) {
	box, _ := envFromDevboxForTesting(t, "base.env", "exec:secrets", "override.env")
	box.cfg.Root.Env = map[string]string{"FROM_CONFIG": "config"}
	writeProjectFile(t, box, "base.env", "PLAIN=base\nPASS=base\nTOKEN=base\n")
	writeProjectFile(t, box, "override.env", "TOKEN=override\nREF=$PASS\n")
	writeMockCommand(t, box, "echo '{\"PASS\":\"secret\",\"TOKEN\":\"secret\"}'\n")

	env, err := box.configEnvs(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Equal(t, "base", env["PLAIN"])
	assert.Equal(t, "secret", env["PASS"])
	assert.Equal(t, "override", env["TOKEN"])
	assert.Equal(t, "secret", env["REF"])
	assert.Equal(t, "config", env["FROM_CONFIG"])
	assert.ElementsMatch(t, []string{"PASS", "REF"}, box.secretEnvKeys)
}

func TestConfigEnvs_ExecFailureKeepsOtherSources(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t, "exec:secrets", "base.env")
	writeProjectFile(t, box, "base.env", "PLAIN=base\n")
	writeMockCommand(t, box, "exit 1\n")

	env, err := box.configEnvs(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Equal(t, "base", env["PLAIN"])
	assert.Empty(t, box.secretEnvKeys)
	assert.Contains(t, stderr.String(), "command failed")
}

func TestConfigEnvs_ExecSeesEarlierSources(t *testing.T) {
	box, _ := envFromDevboxForTesting(t, "keys.env", "exec:secrets", "")
	box.cfg.Root.Env = map[string]string{"KEY_PATH": "$KEY_DIR/key.txt"}
	writeProjectFile(t, box, "keys.env", "KEY_DIR=/keys\n")
	writeMockCommand(t, box, "echo \"{\\\"GOT\\\":\\\"$KEY_PATH\\\"}\"\n")

	env, err := box.configEnvs(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Equal(t, "/keys/key.txt", env["GOT"])
}

func TestExecEnv_SkipsInvalidKeysAndValues(t *testing.T) {
	box, stderr := envFromDevboxForTesting(t)
	writeMockCommand(t, box, "echo '{\"\":\"x\",\"A=B\":\"x\",\"NUL\":\"a\\u0000b\",\"OK\":\"x\"}'\n")

	env := box.execEnv(t.Context(), "secrets", profileEnv(box))
	assert.Equal(t, map[string]string{"OK": "x"}, env)
	assert.Contains(t, stderr.String(), `keys with an invalid name or value: "", "A=B", "NUL".`)
}

func TestConfigEnvs_InvalidSources(t *testing.T) {
	box, _ := envFromDevboxForTesting(t, "exec: ")
	_, err := box.configEnvs(t.Context(), profileEnv(box))
	require.ErrorContains(t, err, "missing the command to run")

	box, _ = envFromDevboxForTesting(t, "base.env", "secrets.yaml")
	writeProjectFile(t, box, "base.env", "PLAIN=base\n")
	_, err = box.configEnvs(t.Context(), profileEnv(box))
	require.ErrorContains(t, err, "unknown env_from value: secrets.yaml")
}

func shellForTesting(t *testing.T, name name, env map[string]string, secretKeys ...string) *DevboxShell {
	t.Helper()
	box, _ := envFromDevboxForTesting(t)
	box.secretEnvKeys = secretKeys
	return &DevboxShell{
		devbox:      box,
		env:         env,
		projectDir:  box.projectDir,
		name:        name,
		interactive: true,
	}
}

func TestExportEnv_Secrets(t *testing.T) {
	env := map[string]string{
		"API_KEY":                 "abc",
		"BASH_FUNC_f%%":           "() { echo secret; }",
		"__DEVBOX_SECRET_INHERIT": "leak",
		"FOO":                     "bar",
	}
	secretKeys := []string{"API_KEY", "MISSING", "BASH_FUNC_f%%"}

	shell := shellForTesting(t, shBash, env, secretKeys...)
	assert.Equal(t,
		"export FOO=\"bar\";\n"+
			`export API_KEY="${__DEVBOX_SECRET_API_KEY}"; unset __DEVBOX_SECRET_API_KEY;`,
		shell.exportEnv())

	shell = shellForTesting(t, shFish, env, secretKeys...)
	assert.Equal(t,
		"export FOO=\"bar\";\n"+
			`export API_KEY="$__DEVBOX_SECRET_API_KEY"; set -e __DEVBOX_SECRET_API_KEY`,
		shell.exportEnv())

	shell = shellForTesting(t, shUnknown, env, secretKeys...)
	assert.Equal(t, `export FOO="bar";`, shell.exportEnv())

	shell = shellForTesting(t, shBash, env, secretKeys...)
	shell.interactive = false
	assert.Equal(t, `export FOO="bar";`, shell.exportEnv())

	shell = shellForTesting(t, shFish, env, secretKeys...)
	shell.interactive = false
	assert.Equal(t,
		"export FOO=\"bar\";\n"+
			`export API_KEY="$__DEVBOX_SECRET_API_KEY"; set -e __DEVBOX_SECRET_API_KEY`,
		shell.exportEnv())
}

func TestExportEnv_WarnsOnInvalidSecretName(t *testing.T) {
	shell := shellForTesting(t, shBash, map[string]string{"my-key": "abc"}, "my-key")
	stderr := &bytes.Buffer{}
	shell.devbox.stderr = stderr

	assert.Empty(t, shell.exportEnv())
	assert.Contains(t, stderr.String(), `invalid names: "my-key"`)
}

func TestProcessEnv_Secrets(t *testing.T) {
	newEnv := func() map[string]string {
		return map[string]string{"API_KEY": "abc", "__DEVBOX_SECRET_OLD": "leak", "FOO": "bar"}
	}

	shell := shellForTesting(t, shBash, newEnv(), "API_KEY")
	shell.binPath = "/bin/bash"
	assert.Equal(t, map[string]string{
		"API_KEY":                 "abc",
		"__DEVBOX_SECRET_API_KEY": "abc",
		"FOO":                     "bar",
		"SHELL":                   "/bin/bash",
		"ENV":                     "/rc",
	}, shell.processEnv(map[string]string{"ENV": "/rc"}))

	shell = shellForTesting(t, shBash, newEnv(), "API_KEY")
	shell.interactive = false
	assert.NotContains(t, shell.processEnv(nil), "__DEVBOX_SECRET_API_KEY")
	assert.NotContains(t, shell.processEnv(nil), "__DEVBOX_SECRET_OLD")
}

func TestWriteDevboxShellrc_SecretsNotOnDisk(t *testing.T) {
	shell := shellForTesting(t, shBash, map[string]string{"API_KEY": "s3cr3t", "FOO": "bar"}, "API_KEY")

	path, err := shell.writeDevboxShellrc()
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(filepath.Dir(path)) })
	shellrc, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.NotContains(t, string(shellrc), "s3cr3t")
	assert.Contains(t, string(shellrc),
		"export FOO=\"bar\";\n"+
			`export API_KEY="${__DEVBOX_SECRET_API_KEY}"; unset __DEVBOX_SECRET_API_KEY;`)
}

func TestAddHashToEnv_IgnoresSecrets(t *testing.T) {
	box, _ := envFromDevboxForTesting(t)
	box.secretEnvKeys = []string{"PASS"}
	hash := func(pass string) string {
		env := map[string]string{"PASS": pass, "FOO": "bar"}
		require.NoError(t, box.addHashToEnv(env))
		assert.Equal(t, pass, env["PASS"])
		return env[box.shellEnvHashKey()]
	}
	assert.Equal(t, hash("one"), hash("two"))
}

func TestMarkPathStackSecret(t *testing.T) {
	box, _ := envFromDevboxForTesting(t)
	box.secretEnvKeys = []string{"PASS"}
	box.markPathStackSecret()
	assert.Equal(t, []string{"PASS"}, box.secretEnvKeys)

	box.secretEnvKeys = []string{"PASS", "PATH"}
	box.markPathStackSecret()
	assert.Equal(t, []string{"PASS", "PATH", envpath.Key(box.ProjectDirHash())}, box.secretEnvKeys)
}
