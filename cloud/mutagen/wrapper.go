package mutagen

import (
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
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

	if spec.IgnoreVCS {
		args = append(args, "--ignore-vcs")
	}

	return execMutagen(args)
}

func List(names ...string) ([]Session, error) {
	binPath := ensureMutagen()
	cmd := exec.Command(binPath, "sync", "list", "--template", "{{json .}}")
	cmd.Args = append(cmd.Args, names...)

	out, err := cmd.CombinedOutput()

	if err != nil {
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

func Resume(names ...string) error {
	args := []string{"sync", "resume"}
	args = append(args, names...)
	return execMutagen(args)
}

func Flush(names ...string) error {
	args := []string{"sync", "flush"}
	args = append(args, names...)
	return execMutagen(args)
}

func Reset(names ...string) error {
	args := []string{"sync", "reset"}
	args = append(args, names...)
	return execMutagen(args)
}

func Terminate(labels map[string]string, names ...string) error {
	args := []string{"sync", "terminate"}

	if len(labels) > 0 {
		var labelSelector string
		for k, v := range labels {
			labelSelector = fmt.Sprintf("%s,%s", labelSelector, fmt.Sprintf("%s=%s", k, v))
		}
		args = append(args, "--label-selector", labelSelector)
	}

	args = append(args, names...)
	return execMutagen(args)
}

func execMutagen(args []string) error {
	binPath := ensureMutagen()
	cmd := exec.Command(binPath, args...)

	out, err := cmd.CombinedOutput()

	if err != nil {
		if e := (&exec.ExitError{}); errors.As(err, &e) {
			return errors.New(strings.TrimSpace(string(out)))
		}
		return err
	}

	return nil
}

func ensureMutagen() string {
	installPath := CacheSubpath("mutagen/bin/mutagen")
	err := InstallMutagenOnce(installPath)
	if err != nil {
		panic(err)
	}
	return installPath
}
