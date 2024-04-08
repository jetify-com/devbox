// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"encoding/json"

	"github.com/pkg/errors"
)

type Forward struct {
	Source struct {
		Connected bool   `json:"connected"`
		Endpoint  string `json:"endpoint"`
	} `json:"source"`
	Destination struct {
		Endpoint string `json:"endpoint"`
	} `json:"destination"`
	LastError string `json:"lastError"`
}

// ForwardCreate creates a new port forward using mutagen.
// local looks like tcp:127.0.0.1:<port>
// remote looks like <host>:<ssh-port>:tcp::<port> (ssh-port is usually 22)
func ForwardCreate(env map[string]string, local, remote string, labels map[string]string) error {
	args := []string{"forward", "create", local, remote}
	return execMutagenEnv(append(args, labelFlag(labels)...), env)
}

func ForwardTerminate(env, labels map[string]string) error {
	args := []string{"forward", "terminate"}
	return execMutagenEnv(append(args, labelSelectorFlag(labels)...), env)
}

func ForwardList(env, labels map[string]string) ([]Forward, error) {
	args := []string{"forward", "list", "--template", "{{json .}}"}
	out, err := execMutagenOut(append(args, labelSelectorFlag(labels)...), env)
	if err != nil {
		return nil, err
	}

	list := []Forward{}
	return list, errors.WithStack(json.Unmarshal(out, &list))
}
