// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/devbox/envpath"
	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/devconfig/configfile"
	"go.jetify.com/devbox/internal/nix"
)

func writeMockSops(t *testing.T, dir, script string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))
	path := filepath.Join(dir, "sops")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"+script), 0o755))
	return path
}

func sopsDevboxForTesting(t *testing.T, envFrom string) (*Devbox, *bytes.Buffer) {
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

func TestParseSopsJSON(t *testing.T) {
	got, nested, err := parseSopsJSON([]byte(
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

func TestParseSopsJSON_SkipsNestedValues(t *testing.T) {
	got, nested, err := parseSopsJSON([]byte(`{"DB":{"HOST":"localhost"},"LIST":["a"],"PASS":"x"}`))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PASS": "x"}, got)
	assert.ElementsMatch(t, []string{"DB", "LIST"}, nested)
}

func TestSopsEnv_WarnsOnNestedValues(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:secrets.json")
	writeMockSops(t, nix.ProfileBinPath(box.projectDir), "echo '{\"DB\":{\"HOST\":\"h\"},\"PASS\":\"x\"}'\n")
	writeProjectFile(t, box, "secrets.json", "{}")

	res, err := box.sopsEnv(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PASS": "x"}, res.env)
	assert.Equal(t, []string{"PASS"}, res.keys)
	assert.Contains(t, stderr.String(), `Ignoring nested values in SOPS file`)
	assert.Contains(t, stderr.String(), "for keys: DB.")
}

func TestSopsKeyNames(t *testing.T) {
	tests := map[string]struct {
		inputType string
		encrypted string
	}{
		"dotenv": {"dotenv", "# comment\nPASS=ENC[AES256_GCM,data:x]\nAPI_KEY=ENC[AES256_GCM,data:y]\nsops_version=3.13.3\n"},
		"json":   {"json", `{"PASS":"ENC[AES256_GCM,data:x]","API_KEY":"ENC[y]","sops":{"version":"3.13.3"}}`},
		"yaml":   {"yaml", "PASS: ENC[AES256_GCM,data:x]\nAPI_KEY: ENC[y]\nsops:\n  version: 3.13.3\n"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			assert.ElementsMatch(t, []string{"PASS", "API_KEY"}, sopsKeyNames([]byte(tt.encrypted), tt.inputType))
		})
	}
}

func TestFindSopsBinary(t *testing.T) {
	first, second := t.TempDir(), t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(first, "sops"), []byte(""), 0o644))
	want := writeMockSops(t, second, "")

	assert.Equal(t, want, findSopsBinary(first+string(filepath.ListSeparator)+second))
	assert.Empty(t, findSopsBinary(first))

	t.Chdir(second)
	assert.Empty(t, findSopsBinary("."))
}

func TestSopsProcessEnv(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	box.cfg.Root.Env = map[string]string{"SOPS_AGE_KEY_FILE": "$PWD/key.txt", "FROM_FLAG": "config"}
	box.env = map[string]string{"FROM_FLAG": "flag"}

	box.cfg.Root.Env["PATH"] = "/repo/tools:$PATH"
	env := box.sopsProcessEnv(map[string]string{"PATH": "/profile/bin", "HOME": "/home/me"})
	assert.Equal(t, filepath.Join(box.projectDir, "key.txt"), env["SOPS_AGE_KEY_FILE"])
	assert.Equal(t, "/home/me", env["HOME"])
	assert.Equal(t, "flag", env["FROM_FLAG"])
	profile, runx := nix.ProfileBinPath(box.projectDir), runxBinPath(box.projectDir)
	t.Setenv("PATH", "/host/bin")
	env = box.sopsProcessEnv(map[string]string{"PATH": "/nix/env/bin"})
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/nix/env/bin", "/host/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])

	// With OmitNixEnv, the computed PATH already contains the host PATH.
	env = box.sopsProcessEnv(map[string]string{"PATH": "/host/bin"})
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/host/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])

	env = box.sopsProcessEnv(map[string]string{"PATH": "/nix/env/bin", "DEVBOX_PURE_SHELL": "1"})
	assert.Equal(t, strings.Join(
		[]string{profile, runx, "/nix/env/bin"},
		string(filepath.ListSeparator),
	), env["PATH"])
}

func TestSopsEnv_RequestsJSONOutput(t *testing.T) {
	for file, inputType := range map[string]string{
		"secrets.yaml": "yaml",
		"secrets.YML":  "yaml",
		"secrets.json": "json",
		"secrets.env":  "dotenv",
	} {
		t.Run(file, func(t *testing.T) {
			box, _ := sopsDevboxForTesting(t, "sops:"+file)
			args := filepath.Join(box.projectDir, "args")
			writeMockSops(t, nix.ProfileBinPath(box.projectDir),
				"echo \"$@\" > '"+args+"'\necho '{\"A\":\"x=\\\"y\\\" # z\"}'\n")
			writeProjectFile(t, box, file, "")

			res, err := box.sopsEnv(t.Context(), profileEnv(box))
			require.NoError(t, err)
			assert.Equal(t, map[string]string{"A": `x="y" # z`}, res.env)
			got, err := os.ReadFile(args)
			require.NoError(t, err)
			path := filepath.Join(box.projectDir, file)
			assert.Equal(t, "decrypt --input-type "+inputType+" --output-type json "+path+"\n", string(got))
		})
	}
}

func TestSopsEnv_RunsInProcessEnv(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:secrets.json")
	writeMockSops(t, nix.ProfileBinPath(box.projectDir),
		"echo 'touch your key' >&2\necho \"{\\\"KEY_FILE\\\":\\\"$SOPS_AGE_KEY_FILE\\\"}\"\n")
	writeProjectFile(t, box, "secrets.json", "{}")
	procEnv := profileEnv(box)
	procEnv["SOPS_AGE_KEY_FILE"] = "/keys.txt"

	res, err := box.sopsEnv(t.Context(), procEnv)
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"KEY_FILE": "/keys.txt"}, res.env)
	assert.Contains(t, stderr.String(), "touch your key")
}

func TestSopsEnv_DecryptsOnce(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	counter := filepath.Join(box.projectDir, "calls")
	writeMockSops(t, nix.ProfileBinPath(box.projectDir),
		"echo call >> '"+counter+"'\necho '{\"API_KEY\":\"abc\"}'\n")
	writeProjectFile(t, box, "secrets.json", "{}")

	for range 2 {
		res, err := box.sopsEnv(t.Context(), profileEnv(box))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"API_KEY": "abc"}, res.env)
	}
	calls, err := os.ReadFile(counter)
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(calls), "call"))
}

func TestSopsEnv_WarnsOnMissingFile(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:missing.json")

	res, err := box.sopsEnv(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Empty(t, res.env)
	assert.Contains(t, stderr.String(), "does not exist")
}

func TestSopsEnv_WarnsOnMissingBinary(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:secrets.json")
	writeProjectFile(t, box, "secrets.json", `{"PASS":"ENC[x]","sops":{}}`)

	res, err := box.sopsEnv(t.Context(), map[string]string{"PATH": t.TempDir()})
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PASS": ""}, res.env)
	assert.Equal(t, []string{"PASS"}, res.keys)
	assert.Contains(t, stderr.String(), "sops binary not found")
	assert.Contains(t, stderr.String(), "Unsetting the variables declared in the SOPS file: PASS.")
}

func TestSopsEnv_WarnsOnceOnDecryptFailure(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:secrets.json")
	writeMockSops(t, nix.ProfileBinPath(box.projectDir), "echo 'no key' >&2\nexit 1\n")
	writeProjectFile(t, box, "secrets.json", `{"PASS":"ENC[x]","sops":{}}`)

	for range 2 {
		res, err := box.sopsEnv(t.Context(), profileEnv(box))
		require.NoError(t, err)
		assert.Equal(t, map[string]string{"PASS": ""}, res.env)
		assert.Equal(t, []string{"PASS"}, res.keys)
	}
	assert.Equal(t, 1, strings.Count(stderr.String(), "no key"))
	assert.Equal(t, 1, strings.Count(stderr.String(), "failed to decrypt SOPS file"))
}

func TestSopsEnv_UnsupportedExtension(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.txt")
	writeProjectFile(t, box, "secrets.txt", "")

	_, err := box.sopsEnv(t.Context(), profileEnv(box))
	require.ErrorContains(t, err, "unsupported SOPS file extension")
}

func TestCollectSecretEnvKeys(t *testing.T) {
	got := collectSecretEnvKeys([]string{"PASS"}, map[string]string{
		"DSN":     "postgres://u:$PASS@h",
		"BRACED":  "${PASS}",
		"PLAIN":   "value",
		"PASSING": "$PASSWORD",
	})
	assert.ElementsMatch(t, []string{"PASS", "DSN", "BRACED"}, got)
}

func TestConfigEnvs_Sops(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	box.cfg.Root.Env = map[string]string{
		"OVERRIDE": "from_config",
		"HOME_DIR": "$HOME",
		"DSN":      "u:$PASS@h",
	}
	writeMockSops(t, nix.ProfileBinPath(box.projectDir),
		"echo '{\"PASS\":\"pa$$word\",\"OVERRIDE\":\"from_sops\"}'\n")
	writeProjectFile(t, box, "secrets.json", "{}")
	existing := profileEnv(box)
	existing["HOME"] = "/home/me"
	existing["PASS"] = "from_host"

	env, err := box.configEnvs(t.Context(), existing)
	require.NoError(t, err)
	assert.Equal(t, "pa$$word", env["PASS"])
	assert.Equal(t, "u:pa$$word@h", env["DSN"])
	assert.Equal(t, "from_config", env["OVERRIDE"])
	assert.Equal(t, "/home/me", env["HOME_DIR"])
	assert.ElementsMatch(t, []string{"PASS", "OVERRIDE", "DSN"}, box.secretEnvKeys)
}

func shellForTesting(t *testing.T, name name, env map[string]string, secretKeys ...string) *DevboxShell {
	t.Helper()
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
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
			`set -gx API_KEY "$__DEVBOX_SECRET_API_KEY"; set -e __DEVBOX_SECRET_API_KEY`,
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
			`set -gx API_KEY "$__DEVBOX_SECRET_API_KEY"; set -e __DEVBOX_SECRET_API_KEY`,
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
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	box.secretEnvKeys = []string{"PASS"}
	hash := func(pass string) string {
		env := map[string]string{"PASS": pass, "FOO": "bar"}
		require.NoError(t, box.addHashToEnv(env))
		assert.Equal(t, pass, env["PASS"])
		return env[box.shellEnvHashKey()]
	}
	assert.Equal(t, hash("one"), hash("two"))
}

func TestSopsEnv_IgnoresPath(t *testing.T) {
	box, stderr := sopsDevboxForTesting(t, "sops:secrets.json")
	writeMockSops(t, nix.ProfileBinPath(box.projectDir), "echo '{\"PATH\":\"/evil\",\"PASS\":\"x\"}'\n")
	writeProjectFile(t, box, "secrets.json", "{}")

	res, err := box.sopsEnv(t.Context(), profileEnv(box))
	require.NoError(t, err)
	assert.Equal(t, map[string]string{"PASS": "x"}, res.env)
	assert.Equal(t, []string{"PASS"}, res.keys)
	assert.Contains(t, stderr.String(), "Ignoring PATH")
}

func TestSopsEnv_IgnoresPathKeyOnFailure(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	writeProjectFile(t, box, "secrets.json", `{"PATH":"ENC[x]","PASS":"ENC[y]","sops":{}}`)

	res, err := box.sopsEnv(t.Context(), map[string]string{"PATH": t.TempDir()})
	require.NoError(t, err)
	assert.Equal(t, []string{"PASS"}, res.keys)
}

func TestMarkPathStackSecret(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	box.secretEnvKeys = []string{"PASS"}
	box.markPathStackSecret()
	assert.Equal(t, []string{"PASS"}, box.secretEnvKeys)

	box.secretEnvKeys = []string{"PASS", "PATH"}
	box.markPathStackSecret()
	assert.Equal(t, []string{"PASS", "PATH", envpath.Key(box.ProjectDirHash())}, box.secretEnvKeys)
}

func TestSopsEnv_EmptyPath(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:")

	_, err := box.sopsEnv(t.Context(), profileEnv(box))
	require.ErrorContains(t, err, "missing the path to the SOPS file")
}

func TestConfigEnvs_SopsFailureUnsetsHostValues(t *testing.T) {
	box, _ := sopsDevboxForTesting(t, "sops:secrets.json")
	box.cfg.Root.Env = map[string]string{"DSN": "u:$PASS@h", "OVERRIDE": "from_config"}
	writeMockSops(t, nix.ProfileBinPath(box.projectDir), "exit 1\n")
	writeProjectFile(t, box, "secrets.json", `{"PASS":"ENC[x]","OVERRIDE":"ENC[y]","sops":{}}`)
	existing := profileEnv(box)
	existing["PASS"] = "from_host"

	env, err := box.configEnvs(t.Context(), existing)
	require.NoError(t, err)
	assert.Equal(t, "", env["PASS"])
	assert.Contains(t, env, "PASS")
	assert.Equal(t, "u:@h", env["DSN"])
	assert.Equal(t, "from_config", env["OVERRIDE"])
}
