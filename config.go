// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
)

// Config defines a devbox environment as JSON.
type Config struct {
	// Packages is the slice of Nix packages that devbox makes available in
	// its environment.
	Packages []string `cue:"[...string]" json:"packages"`
	// InstallStage defines the actions that should be taken when
	// installing language-specific libraries.
	InstallStage *Stage `json:"install_stage,omitempty"`
	// BuildStage defines the actions that should be taken when
	// compiling the application binary.
	BuildStage *Stage `json:"build_stage,omitempty"`
	// StartStage defines the actions that should be taken when
	// starting (running) the application.
	StartStage *Stage `json:"start_stage,omitempty"`

	// Shell configures the devbox shell environment.
	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook ConfigShellCmds `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`
}

// This contains a subset of fields from plansdk.Stage
type Stage struct {
	Command string `cue:"string" json:"command"`
}

// ReadConfig reads a devbox config file.
func ReadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := cuecfg.ParseFile(path, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

// WriteConfig saves a devbox config file.
func WriteConfig(path string, cfg *Config) error {
	return cuecfg.WriteFile(path, cfg)
}

// Formats for marshalling and unmarshalling a series of shell commands in a
// devbox config.
const (
	// CmdArray formats shell commands as an array of of strings.
	CmdArray CmdFormat = iota

	// CmdString formats shell commands as a single string.
	CmdString
)

// CmdFormat defines a way of formatting shell commands in a devbox config.
type CmdFormat int

func (c CmdFormat) String() string {
	switch c {
	case CmdArray:
		return "array"
	case CmdString:
		return "string"
	default:
		return fmt.Sprintf("invalid (%d)", c)
	}
}

// ConfigShellCmds marshals and unmarshals shell commands from a devbox config
// as either a single string or an array of strings. It preserves the original
// value such that:
//
//	data == marshal(unmarshal(data)))
type ConfigShellCmds struct {
	// MarshalAs determines how MarshalJSON encodes the shell commands.
	// UnmarshalJSON will set MarshalAs automatically so that commands
	// marshal back to their original format. The default zero-value
	// formats them as an array.
	//
	MarshalAs CmdFormat
	Cmds      []string
}

// MarshalJSON marshals shell commands according to s.MarshalAs. It marshals
// commands to a string by joining s.Cmds with newlines.
func (s ConfigShellCmds) MarshalJSON() ([]byte, error) {
	switch s.MarshalAs {
	case CmdArray:
		return json.Marshal(s.Cmds)
	case CmdString:
		return json.Marshal(s.String())
	default:
		panic(fmt.Sprintf("invalid command format: %s", s.MarshalAs))
	}
}

// UnmarshalJSON unmarshals shell commands from a string, an array of strings,
// or null. When the JSON value is a string, it unmarshals into the first index
// of s.Cmds.
func (s *ConfigShellCmds) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		s.MarshalAs = CmdArray
		s.Cmds = nil
		return nil
	}

	switch data[0] {
	case '"':
		s.MarshalAs = CmdString
		s.Cmds = []string{""}
		return json.Unmarshal(data, &s.Cmds[0])

	case '[':
		s.MarshalAs = CmdArray
		return json.Unmarshal(data, &s.Cmds)
	default:
		return nil
	}
}

// String formats the commands as a single string by joining them with newlines.
func (s *ConfigShellCmds) String() string {
	return strings.Join(s.Cmds, "\n")
}
