// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/conf"
	"go.jetify.com/devbox/internal/devbox/envpath"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/ux"
	"gopkg.in/yaml.v3"
)

type sopsResult struct {
	path string
	env  map[string]string
	// keys are the variable names declared in the file. SOPS leaves them in
	// cleartext, so they are known even when decryption fails.
	keys []string
}

// sopsEnv decrypts the env_from SOPS file at most once per Devbox instance, as
// each decryption may hit a KMS or require a hardware key touch. Failures are
// cached too so that the warning is only printed once.
func (d *Devbox) sopsEnv(ctx context.Context, procEnv map[string]string) (*sopsResult, error) {
	if strings.TrimSpace(strings.TrimPrefix(d.cfg.Root.EnvFrom, "sops:")) == "" {
		return nil, usererr.New("env_from %q is missing the path to the SOPS file", d.cfg.Root.EnvFrom)
	}
	filePath := d.cfg.Root.SopsFilePath()
	if d.sopsCache != nil && d.sopsCache.path == filePath {
		return d.sopsCache, nil
	}
	res, err := d.sopsDecrypt(ctx, filePath, procEnv)
	if err != nil {
		return nil, err
	}
	d.sopsCache = res
	return res, nil
}

// sopsProcessEnv is the environment sops runs in: the project's environment,
// so that the sops binary, its helpers (gpg, age plugins) and variables such as
// SOPS_AGE_KEY_FILE set in devbox.json are the project's ones. A PATH set in
// devbox.json env is not used, so that env_from never runs a sops from the
// repository itself.
func (d *Devbox) sopsProcessEnv(existingEnv map[string]string) map[string]string {
	env := make(map[string]string, len(existingEnv))
	maps.Copy(env, existingEnv)
	maps.Copy(env, conf.OSExpandEnvMap(d.cfg.Env(), existingEnv, d.ProjectDir()))
	maps.Copy(env, d.env)
	env["PATH"] = envpath.JoinPathLists(existingEnv["PATH"], runxBinPath(d.projectDir))
	// Outside of --pure, the final devbox PATH gets the host PATH appended
	// after configEnvs.
	if existingEnv["DEVBOX_PURE_SHELL"] != "1" {
		env["PATH"] = envpath.JoinPathLists(env["PATH"], os.Getenv("PATH"))
	}
	return env
}

func (d *Devbox) sopsDecrypt(
	ctx context.Context,
	filePath string,
	procEnv map[string]string,
) (*sopsResult, error) {
	res := &sopsResult{path: filePath}

	var inputType string
	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".yaml", ".yml":
		inputType = "yaml"
	case ".json":
		inputType = "json"
	case ".env":
		inputType = "dotenv"
	default:
		return nil, usererr.New(
			"unsupported SOPS file extension %q. Supported extensions: .yaml, .yml, .json, .env",
			ext,
		)
	}

	encrypted, err := os.ReadFile(filePath)
	if errors.Is(err, os.ErrNotExist) {
		ux.Fwarningf(
			d.stderr,
			"Ignoring env_from directive: SOPS file %q does not exist.\n",
			filePath,
		)
		return res, nil
	} else if err != nil {
		ux.Fwarningf(d.stderr, "Ignoring env_from directive: %v\n", err)
		return res, nil
	}
	res.keys = slices.DeleteFunc(sopsKeyNames(encrypted, inputType), isPathKey)

	sopsBin := findSopsBinary(procEnv["PATH"])
	if sopsBin == "" {
		ux.Fwarningf(
			d.stderr,
			"Ignoring env_from directive: sops binary not found. "+
				"Run `devbox add sops` to add it to your project.\n",
		)
		return res.unset(d), nil
	}

	cmd := exec.CommandContext(
		ctx, sopsBin, "decrypt",
		// Always ask for JSON so that every input format goes through the same
		// parser: sops' dotenv output is raw KEY=VALUE lines that .env parsers
		// would misread (quotes, #, escaped newlines).
		"--input-type", inputType, "--output-type", "json",
		filePath,
	)
	cmd.Env = envir.MapToPairs(procEnv)
	// sops may need to prompt (PIN, hardware key touch), so it gets the
	// terminal and its messages are shown as they come.
	cmd.Stdin = os.Stdin
	cmd.Stderr = d.stderr
	out, err := cmd.Output()
	if err != nil {
		ux.Fwarningf(
			d.stderr,
			"Ignoring env_from directive: failed to decrypt SOPS file %q: %v\n",
			filePath, err,
		)
		return res.unset(d), nil
	}

	var nested []string
	res.env, nested, err = parseSopsJSON(out)
	if err != nil {
		return nil, err
	}
	if len(nested) > 0 {
		slices.Sort(nested)
		ux.Fwarningf(
			d.stderr,
			"Ignoring nested values in SOPS file %q for keys: %s. "+
				"Only flat key-value pairs are supported for env_from.\n",
			filePath, strings.Join(nested, ", "),
		)
	}
	if _, ok := res.env["PATH"]; ok {
		ux.Fwarningf(
			d.stderr,
			"Ignoring PATH from SOPS file %q: it would replace the devbox PATH.\n",
			filePath,
		)
		delete(res.env, "PATH")
	}
	res.keys = slices.Collect(maps.Keys(res.env))
	return res, nil
}

// unset empties the variables declared in the file when it can't be
// decrypted, so that commands don't silently run with the values inherited
// from the host.
func (r *sopsResult) unset(d *Devbox) *sopsResult {
	if len(r.keys) == 0 {
		return r
	}
	r.env = make(map[string]string, len(r.keys))
	for _, k := range r.keys {
		r.env[k] = ""
	}
	slices.Sort(r.keys)
	ux.Fwarningf(d.stderr, "Unsetting the variables declared in the SOPS file: %s.\n", strings.Join(r.keys, ", "))
	return r
}

// markPathStackSecret marks the path stack's copy of PATH as secret when PATH
// references a secret.
func (d *Devbox) markPathStackSecret() {
	if slices.Contains(d.secretEnvKeys, "PATH") {
		d.secretEnvKeys = append(d.secretEnvKeys, envpath.Key(d.ProjectDirHash()))
	}
}

func isPathKey(key string) bool {
	return key == "PATH"
}

func runxBinPath(projectDir string) string {
	return filepath.Join(projectDir, ".devbox", "virtenv", "runx", "bin")
}

func findSopsBinary(pathList string) string {
	for _, dir := range filepath.SplitList(pathList) {
		if !filepath.IsAbs(dir) {
			continue
		}
		path := filepath.Join(dir, "sops")
		if fi, err := os.Stat(path); err == nil && !fi.IsDir() && fi.Mode()&0o111 != 0 {
			return path
		}
	}
	return ""
}

func sopsKeyNames(encrypted []byte, inputType string) []string {
	var keys []string
	switch inputType {
	case "dotenv":
		for line := range strings.SplitSeq(string(encrypted), "\n") {
			key, _, ok := strings.Cut(strings.TrimSpace(line), "=")
			if ok && !strings.HasPrefix(key, "#") && !strings.HasPrefix(key, "sops_") {
				keys = append(keys, key)
			}
		}
		return keys
	case "json":
		var doc map[string]json.RawMessage
		if json.Unmarshal(encrypted, &doc) != nil {
			return nil
		}
		for k := range doc {
			keys = append(keys, k)
		}
	default:
		var doc map[string]any
		if yaml.Unmarshal(encrypted, &doc) != nil {
			return nil
		}
		for k := range doc {
			keys = append(keys, k)
		}
	}
	return slices.DeleteFunc(keys, func(k string) bool { return k == "sops" })
}

// collectSecretEnvKeys returns the SOPS keys plus the devbox.json env variables whose
// value references one of them, as those embed the secret once expanded.
func collectSecretEnvKeys(sopsKeys []string, cfgEnv map[string]string) []string {
	secret := make(map[string]bool, len(sopsKeys))
	for _, k := range sopsKeys {
		secret[k] = true
	}
	keys := slices.Clone(sopsKeys)
	for k, v := range cfgEnv {
		if secret[k] {
			continue
		}
		references := false
		os.Expand(v, func(name string) string {
			references = references || secret[name]
			return ""
		})
		if references {
			keys = append(keys, k)
		}
	}
	return keys
}

// parseSopsJSON returns the flat values and the keys holding nested values,
// which are skipped.
func parseSopsJSON(data []byte) (env map[string]string, nested []string, err error) {
	var raw map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, nil, usererr.New(
			"failed to parse decrypted SOPS output as JSON: %v", err,
		)
	}

	result := make(map[string]string, len(raw))
	for key, v := range raw {
		switch val := v.(type) {
		case string:
			result[key] = val
		case json.Number:
			result[key] = val.String()
		case bool:
			result[key] = strconv.FormatBool(val)
		case nil:
			result[key] = ""
		default:
			nested = append(nested, key)
		}
	}
	return result, nested, nil
}
