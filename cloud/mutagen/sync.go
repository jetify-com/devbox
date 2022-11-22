package mutagen

import (
	"errors"
)

func Sync(spec *SessionSpec) (*Session, error) {
	if spec.Name == "" {
		return nil, errors.New("name is required")
	}

	// Check if there's an existing sessions or not
	sessions, err := List(spec.Name)
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
	sessions, err = List(spec.Name)
	if err != nil {
		return nil, err
	}
	for _, session := range sessions {
		Reset(session.Identifier)
		Resume(session.Identifier)
	}
	if len(sessions) > 0 {
		return &sessions[0], nil
	} else {
		return nil, errors.New("failed to find session that was just created")
	}
	// TODO: starting the mutagen session currently fails if there's any error or
	// interactivity required for the ssh connection.
	// That includes:
	// - When connecting for the first time and adding the host to known_hosts
	// - When the key has changed and SSH warns of a man-in-the-middle attack
}
