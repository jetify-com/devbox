// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"errors"
)

func Sync(spec *SessionSpec) (*Session, error) {
	if spec.Name == "" {
		return nil, errors.New("name is required")
	}

	// Check if there's an existing sessions or not
	sessions, err := List(spec.EnvVars, spec.Name)
	if err != nil {
		return nil, err
	}

	// If there isn't, create a new one
	if len(sessions) == 0 {
		err = Create(spec)
		if err != nil {
			return nil, err
		}
	}
	// Whether new or pre-existing, find the sessions object, ensure
	// that it's not paused, and return it.
	sessions, err = List(spec.EnvVars, spec.Name)
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		// TODO: should we handle errors for Reset and Resume differently?
		_ = Reset(spec.EnvVars, session.Identifier)
		_ = Resume(spec.EnvVars, session.Identifier)
	}
	if len(sessions) > 0 {
		return &sessions[0], nil
	}
	return nil, errors.New("failed to find session that was just created")
	// TODO: starting the mutagen session currently fails if there's any error or
	// interactivity required for the ssh connection.
	// That includes:
	// - When connecting for the first time and adding the host to known_hosts
	// - When the key has changed and SSH warns of a man-in-the-middle attack
}
