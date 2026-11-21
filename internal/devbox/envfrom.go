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
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"go.jetify.com/devbox/internal/conf"
	"go.jetify.com/devbox/internal/devbox/envpath"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/nix"
	"go.jetify.com/devbox/internal/ux"
)

// execEnv runs each command once per Devbox instance, failures included, as it
// may hit a secret manager or require a hardware key touch.
func (d *Devbox) execEnv(ctx context.Context, command string, procEnv map[string]string) map[string]string {
	if env, ok := d.execCache[command]; ok {
		return env
	}
	env := d.runExecCommand(ctx, command, procEnv)
	if d.execCache == nil {
		d.execCache = map[string]map[string]string{}
	}
	d.execCache[command] = env
	return env
}

// execProcessEnv is the project's environment, including the earlier env_from
// sources. Its PATH ignores the PATH set in devbox.json and starts with the
// profile and runx, as with OmitNixEnv existingEnv's PATH already contains the
// host PATH.
func (d *Devbox) execProcessEnv(existingEnv, sourcesEnv map[string]string) map[string]string {
	env := make(map[string]string, len(existingEnv)+len(sourcesEnv))
	maps.Copy(env, existingEnv)
	maps.Copy(env, sourcesEnv)
	maps.Copy(env, conf.OSExpandEnvMap(d.cfg.Env(), env, d.ProjectDir()))
	maps.Copy(env, d.env)
	env["PATH"] = envpath.JoinPathLists(
		nix.ProfileBinPath(d.projectDir),
		runxBinPath(d.projectDir),
		existingEnv["PATH"],
	)
	if existingEnv["DEVBOX_PURE_SHELL"] != "1" {
		env["PATH"] = envpath.JoinPathLists(env["PATH"], os.Getenv("PATH"))
	}
	return env
}

func (d *Devbox) runExecCommand(ctx context.Context, command string, procEnv map[string]string) map[string]string {
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	cmd.Env = envir.MapToPairs(procEnv)
	cmd.Dir = d.projectDir
	// The command may prompt (PIN, hardware key touch), but piped input
	// belongs to the devbox command.
	if isatty.IsTerminal(os.Stdin.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stderr = d.stderr
	// Don't wait for background processes holding stdout, such as agents.
	cmd.WaitDelay = time.Second
	out, err := cmd.Output()
	if err != nil && !errors.Is(err, exec.ErrWaitDelay) {
		ux.Fwarningf(d.stderr, "Ignoring env_from %q: command failed: %v\n", "exec:"+command, err)
		return nil
	}

	env, nested, err := parseEnvJSON(out)
	if err != nil {
		ux.Fwarningf(
			d.stderr,
			"Ignoring env_from %q: output is not a flat JSON object: %v\n",
			"exec:"+command, err,
		)
		return nil
	}
	if len(nested) > 0 {
		slices.Sort(nested)
		ux.Fwarningf(
			d.stderr,
			"Ignoring nested values from env_from %q for keys: %s. "+
				"Only flat key-value pairs are supported.\n",
			"exec:"+command, strings.Join(nested, ", "),
		)
	}
	var invalid []string
	for k, v := range env {
		if !isValidEnvName(k) || strings.ContainsRune(v, 0) {
			invalid = append(invalid, strconv.Quote(k))
			delete(env, k)
		}
	}
	if len(invalid) > 0 {
		slices.Sort(invalid)
		ux.Fwarningf(
			d.stderr,
			"Ignoring env_from %q keys with an invalid name or value: %s.\n",
			"exec:"+command, strings.Join(invalid, ", "),
		)
	}
	if _, ok := env["PATH"]; ok {
		ux.Fwarningf(
			d.stderr,
			"Ignoring PATH from env_from %q: it would replace the devbox PATH.\n",
			"exec:"+command,
		)
		delete(env, "PATH")
	}
	return env
}

func (d *Devbox) markPathStackSecret() {
	if slices.Contains(d.secretEnvKeys, "PATH") {
		d.secretEnvKeys = append(d.secretEnvKeys, envpath.Key(d.ProjectDirHash()))
	}
}

// collectSecretEnvKeys returns the secret keys missing from env, which holds
// the .env and devbox.json values, plus the env variables that embed a secret
// once expanded.
func collectSecretEnvKeys(secretKeys []string, env map[string]string) []string {
	secret := make(map[string]bool, len(secretKeys))
	for _, k := range secretKeys {
		secret[k] = true
	}
	var keys []string
	for k, v := range env {
		references := false
		os.Expand(v, func(name string) string {
			references = references || secret[name]
			return ""
		})
		if references {
			keys = append(keys, k)
		}
	}
	for _, k := range secretKeys {
		if _, ok := env[k]; !ok {
			keys = append(keys, k)
		}
	}
	return keys
}

func parseEnvJSON(data []byte) (env map[string]string, nested []string, err error) {
	var raw map[string]any
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.UseNumber()
	if err := dec.Decode(&raw); err != nil {
		return nil, nil, err
	}
	if dec.More() {
		return nil, nil, errors.New("unexpected data after the JSON object")
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
