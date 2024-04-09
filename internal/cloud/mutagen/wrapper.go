// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/xdg"
)

func Create(spec *SessionSpec) error {
	err := spec.Validate()
	if err != nil {
		return err
	}

	alpha := spec.AlphaPath
	if spec.AlphaAddress != "" {
		alpha = fmt.Sprintf("%s:%s", spec.AlphaAddress, spec.AlphaPath)
	}

	beta := spec.BetaPath
	if spec.BetaAddress != "" {
		beta = fmt.Sprintf("%s:%s", spec.BetaAddress, spec.BetaPath)
	}

	args := []string{"sync", "create", alpha, beta}
	if spec.Name != "" {
		args = append(args, "--name", spec.Name)
	}
	if spec.Paused {
		args = append(args, "--paused")
	}

	for k, v := range spec.Labels {
		args = append(args, "--label", fmt.Sprintf("%s=%s", k, v))
	}

	if spec.SyncMode == "" {
		args = append(args, "--sync-mode", "two-way-resolved")
	} else {
		args = append(args, "--sync-mode", spec.SyncMode)
	}

	if spec.Ignore.VCS {
		args = append(args, "--ignore-vcs")
	}

	if len(spec.Ignore.Paths) > 0 {
		for _, p := range spec.Ignore.Paths {
			args = append(args, "--ignore", p)
		}
	}

	return execMutagenEnv(args, spec.EnvVars)
}

func List(envVars map[string]string, names ...string) ([]Session, error) {
	binPath := ensureMutagen()
	cmd := exec.Command(binPath, "sync", "list", "--template", "{{json .}}")
	cmd.Args = append(cmd.Args, names...)
	cmd.Env = envAsKeyValueStrings(envVars)

	debugPrintExecCmd(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		debug.Log("List error: %s, and out: %s", err, string(out))
		if e := (&exec.ExitError{}); errors.As(err, &e) {
			errMsg := strings.TrimSpace(string(out))
			// Special handle the case where no sessions are found:
			if strings.Contains(errMsg, "unable to locate requested sessions") {
				return []Session{}, nil
			}
			return nil, errors.New(errMsg)
		}
		return nil, err
	}

	sessions := []Session{}
	err = json.Unmarshal(out, &sessions)
	if err != nil {
		return nil, err
	}

	return sessions, nil
}

func Pause(names ...string) error {
	args := []string{"sync", "pause"}
	args = append(args, names...)
	return execMutagen(args)
}

func Resume(envVars map[string]string, names ...string) error {
	args := []string{"sync", "resume"}
	args = append(args, names...)
	return execMutagenEnv(args, envVars)
}

func Flush(names ...string) error {
	args := []string{"sync", "flush"}
	args = append(args, names...)
	return execMutagen(args)
}

func Reset(envVars map[string]string, names ...string) error {
	args := []string{"sync", "reset"}
	args = append(args, names...)
	return execMutagenEnv(args, envVars)
}

func Terminate(env, labels map[string]string, names ...string) error {
	args := []string{"sync", "terminate"}

	for k, v := range labels {
		args = append(args, "--label-selector", fmt.Sprintf("%s=%s", k, v))
	}

	args = append(args, names...)
	return execMutagenEnv(args, env)
}

func execMutagen(args []string) error {
	return execMutagenEnv(args, nil)
}

func execMutagenEnv(args []string, envVars map[string]string) error {
	_, err := execMutagenOut(args, envVars)
	return err
}

func execMutagenOut(args []string, envVars map[string]string) ([]byte, error) {
	binPath := ensureMutagen()
	cmd := exec.Command(binPath, args...)
	cmd.Env = envAsKeyValueStrings(envVars)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	debugPrintExecCmd(cmd)

	if err := cmd.Run(); err != nil {
		debug.Log(
			"execMutagen error: %s, stdout: %s, stderr: %s",
			err,
			stdout.String(),
			stderr.String(),
		)
		if e := (&exec.ExitError{}); errors.As(err, &e) {
			return nil, errors.New(strings.TrimSpace(stderr.String()))
		}
		return nil, err
	}

	debug.Log("execMutagen worked for cmd: %s", cmd)
	return stdout.Bytes(), nil
}

// debugPrintExecCmd prints the command to be run, along with MUTAGEN env-vars
func debugPrintExecCmd(cmd *exec.Cmd) {
	envPrint := ""
	for _, cmdEnv := range cmd.Env {
		if strings.HasPrefix(cmdEnv, "MUTAGEN") {
			envPrint = fmt.Sprintf("%s, %s", envPrint, cmdEnv)
		}
	}
	debug.Log("running mutagen cmd %s with MUTAGEN env: %s", cmd.String(), envPrint)
}

// envAsKeyValueStrings prepares the env-vars in key=value format to add to the command to be run
//
// panics if os.Environ() returns an array with any element not in key=value format
func envAsKeyValueStrings(userEnv map[string]string) []string {
	if userEnv == nil {
		userEnv = map[string]string{}
	}

	// Convert env to map, and strip out MUTAGEN_PROMPTER env-var
	envMap := map[string]string{}
	for _, envVar := range os.Environ() {
		k, v, found := strings.Cut(envVar, "=")
		if !found {
			panic(fmt.Sprintf("did not find an = in env-var: %s", envVar))
		}
		// Mutagen sets this variable for ssh/scp scenarios, which then expect interactivity?
		// https://github.com/mutagen-io/mutagen/blob/b97ff3764a6a6cb91b48ad27def078f6d6a76e24/cmd/mutagen/main.go#L89-L94
		//
		// We do not include MUTAGEN_PROMPTER, otherwise mutagen-CLI rejects the command we are about to invoke,
		// by treating it instead as a prompter-command.
		if k != "MUTAGEN_PROMPTER" {
			envMap[k] = v
		}
	}

	// userEnv overrides the default env
	for k, v := range userEnv {
		envMap[k] = v
	}

	// Convert the envMap to an envList
	envList := make([]string, 0, len(envMap))
	for k, v := range envMap {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	return envList
}

func ensureMutagen() string {
	installPath := xdg.CacheSubpath("mutagen/bin/mutagen")
	err := InstallMutagenOnce(installPath)
	if err != nil {
		panic(err)
	}
	return installPath
}

func labelFlag(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}
	labelSlice := []string{}
	for k, v := range labels {
		labelSlice = append(labelSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return []string{"--label", strings.Join(labelSlice, ",")}
}

func labelSelectorFlag(labels map[string]string) []string {
	if len(labels) == 0 {
		return []string{}
	}
	labelSlice := []string{}
	for k, v := range labels {
		labelSlice = append(labelSlice, fmt.Sprintf("%s=%s", k, v))
	}
	return []string{"--label-selector", strings.Join(labelSlice, ",")}
}
