// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"io"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

const errApplyNixDerivationString = "Error: apply Nix derivation:"

var sshSessionErrorStrings = []string{
	errApplyNixDerivationString,
}

// sshSessionErrors is a helper struct to collect errors from ssh sessions.
// For performance and privacy it doesn't actually keep any content from the
// sessions, but instead just keeps track of which errors were encountered.
type sshSessionErrors struct {
	errors map[string]bool
}

var _ io.Writer = (*sshSessionErrors)(nil)

func newSSHSessionErrors() *sshSessionErrors {
	return &sshSessionErrors{
		errors: make(map[string]bool),
	}
}

func (s *sshSessionErrors) Write(p []byte) (n int, err error) {
	for _, errorString := range sshSessionErrorStrings {
		if strings.Contains(string(p), errorString) {
			s.errors[errorString] = true
		}
	}
	return len(p), nil
}

// cloudShellErrorHandler is a helper function to handle ssh errors that
// may contain nix errors in them. For now being cautious and logging them
// to Sentry even though they may be due to user action.
func cloudShellErrorHandler(err error, sessionErrors *sshSessionErrors) error {
	if err == nil {
		return nil
	}

	// This usually on initial setup when running start_devbox_shell.sh
	if found := sessionErrors.errors[errApplyNixDerivationString]; found {
		return usererr.WithLoggedUserMessage(
			err,
			"Failed to apply Nix derivation. This can happen if your devbox (nix) "+
				"packages don't exist or failed to build. Please check your "+
				"devbox.json and try again",
		)
	}

	// This can happen due to connection issues or any other unforeseen errors
	return usererr.WithLoggedUserMessage(
		err,
		"Your cloud shell terminated unexpectedly. Please check your connection "+
			"and devbox.json and try again",
	)
}
