// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellcmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"unicode"

	"go.jetpack.io/devbox/internal/cuecfg"
)

// Formats for marshalling and unmarshalling a series of shell commands in a
// devbox config.
const (
	// CmdArray formats shell commands as an array of strings.
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

// Commands marshals and unmarshals shell commands from a devbox config
// as either a single string or an array of strings. It preserves the original
// value such that:
//
//	data == marshal(unmarshal(data)))
type Commands struct {
	// MarshalAs determines how MarshalJSON encodes the shell commands.
	// UnmarshalJSON will set MarshalAs automatically so that commands
	// marshal back to their original format. The default zero-value
	// formats them as an array.
	MarshalAs CmdFormat
	Cmds      []string
}

// AppendScript appends each line of a script to s.Cmds. It also applies the
// following formatting rules:
//
//   - Trim leading newlines from the script.
//   - Trim trailing whitespace from the script.
//   - If the first line of the script begins with one or more tabs ('\t'), then
//     unindent each line by that same number of tabs.
//   - Remove trailing whitespace from each line.
//
// Note that unindenting only happens when a line starts with at least as many
// tabs as the first line. If it starts with fewer tabs, then it is not
// unindented at all.
func (s *Commands) AppendScript(script string) {
	script = strings.TrimLeft(script, "\r\n ")
	script = strings.TrimRightFunc(script, unicode.IsSpace)
	if len(script) == 0 {
		return
	}
	prefixLen := strings.IndexFunc(script, func(r rune) bool { return r != '\t' })
	prefix := strings.Repeat("\t", prefixLen)
	for _, line := range strings.Split(script, "\n") {
		line = strings.TrimRightFunc(line, unicode.IsSpace)
		line = strings.TrimPrefix(line, prefix)
		s.Cmds = append(s.Cmds, line)
	}
}

// MarshalJSON marshals shell commands according to s.MarshalAs. It marshals
// commands to a string by joining s.Cmds with newlines.
func (s Commands) MarshalJSON() ([]byte, error) {
	switch s.MarshalAs {
	case CmdArray:
		return cuecfg.MarshalJSON(s.Cmds)
	case CmdString:
		return cuecfg.MarshalJSON(s.String())
	default:
		panic(fmt.Sprintf("invalid command format: %s", s.MarshalAs))
	}
}

// UnmarshalJSON unmarshals shell commands from a string, an array of strings,
// or null. When the JSON value is a string, it unmarshals into the first index
// of s.Cmds.
func (s *Commands) UnmarshalJSON(data []byte) error {
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
func (s *Commands) String() string {
	if s == nil {
		return ""
	}
	return strings.Join(s.Cmds, "\n")
}
