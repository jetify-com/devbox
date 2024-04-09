// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"errors"
	"unicode"
	"unicode/utf8"
)

type SessionIgnore struct {
	VCS   bool
	Paths []string
}

type SessionSpec struct {
	AlphaAddress string
	AlphaPath    string
	BetaAddress  string
	BetaPath     string
	Name         string
	Labels       map[string]string
	Paused       bool
	SyncMode     string
	Ignore       SessionIgnore
	EnvVars      map[string]string
}

func (s *SessionSpec) Validate() error {
	if s.AlphaPath == "" {
		return errors.New("alpha path is required")
	}
	if s.BetaPath == "" {
		return errors.New("beta path is required")
	}
	return nil
}

// SanitizeSessionName ensures the input string contains letter, number or dash
// runes. This matches the implementation in mutagen's codebase:
// https://github.com/mutagen-io/mutagen/blob/master/pkg/selection/names.go
//
// TODO savil. Refactor SessionSpec so that this is always applied.
// We can make it a struct that uses a constructor, and make Sync a method on the struct.
func SanitizeSessionName(input string) string {
	result := make([]byte, 0, len(input))

	// note that for-range over a string extracts characters of type rune
	for index, char := range input {
		// the first character must be a letter
		if index == 0 && !unicode.IsLetter(char) {
			result = utf8.AppendRune(result, 'a')
		}

		if unicode.IsLetter(char) || unicode.IsNumber(char) || char == '-' {
			result = utf8.AppendRune(result, char)
		} else {
			result = utf8.AppendRune(result, '-')
		}
	}
	return string(result)
}

// Based on the structs available at: https://github.com/mutagen-io/mutagen/blob/master/pkg/api/models/synchronization/session.go
// These contain a subset of fields.

type Session struct {
	Identifier      string            `json:"identifier"`
	Version         uint32            `json:"version"`
	CreationTime    string            `json:"creationTime"`
	CreatingVersion string            `json:"creatingVersion"`
	Alpha           Endpoint          `json:"alpha"`
	Beta            Endpoint          `json:"beta"`
	Name            string            `json:"name,omitempty"`
	Labels          map[string]string `json:"labels,omitempty"`
	Paused          bool              `json:"paused"`
	Status          string            `json:"status"`
}

type Endpoint struct {
	User        string            `json:"user,omitempty"`
	Host        string            `json:"host,omitempty"`
	Port        uint16            `json:"port,omitempty"`
	Path        string            `json:"path"`
	Environment map[string]string `json:"environment,omitempty"`
	Parameters  map[string]string `json:"parameters,omitempty"`
	Connected   bool              `json:"connected"`
}
