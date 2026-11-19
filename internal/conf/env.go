// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package conf

import (
	"strings"
)

// SplitDefault splits a variable reference as passed to an Expand mapping
// function into its name and default value. It supports the shell-style
// "NAME:-default" syntax (as in ${NAME:-default}), where default is used if
// NAME is unset or empty. ok is false if the reference has no default.
func SplitDefault(ref string) (name, def string, ok bool) {
	return strings.Cut(ref, ":-")
}

// Expand is like os.Expand, except that ${...} references are terminated by
// the matching closing brace rather than the first one. This allows defaults
// that themselves contain braced references, e.g. ${FOO:-${BAR}}. The mapping
// function receives everything between the outer braces (e.g. "FOO:-${BAR}").
func Expand(str string, mapping func(string) string) string {
	var buf []byte
	// ${} is all ASCII, so bytes are fine for this operation.
	i := 0
	for pos := 0; pos < len(str); pos++ {
		if str[pos] == '$' && pos+1 < len(str) {
			if buf == nil {
				buf = make([]byte, 0, 2*len(str))
			}
			buf = append(buf, str[i:pos]...)
			name, w := getShellName(str[pos+1:])
			if name == "" && w > 0 {
				// Encountered invalid syntax; eat the characters.
			} else if name == "" {
				// Valid syntax, but $ was not followed by a name. Leave the
				// dollar character untouched.
				buf = append(buf, str[pos])
			} else {
				buf = append(buf, mapping(name)...)
			}
			pos += w
			i = pos + 1
		}
	}
	if buf == nil {
		return str
	}
	return string(buf) + str[i:]
}

// getShellName is adapted from the Go standard library's os package. It
// returns the name that begins the string and the number of bytes consumed
// to extract it. If the name is enclosed in {}, it's part of a ${} expansion
// and two more bytes are needed than the length of the name. Unlike the
// standard library, nested braces are matched.
func getShellName(str string) (string, int) {
	switch {
	case str[0] == '{':
		if len(str) > 2 && isShellSpecialVar(str[1]) && str[2] == '}' {
			return str[1:2], 3
		}
		// Scan to the matching closing brace.
		depth := 0
		for i := 1; i < len(str); i++ {
			switch str[i] {
			case '{':
				depth++
			case '}':
				if depth > 0 {
					depth--
					continue
				}
				if i == 1 {
					return "", 2 // Bad syntax; eat "${}"
				}
				return str[1:i], i + 1
			}
		}
		return "", 1 // Bad syntax; eat "${"
	case isShellSpecialVar(str[0]):
		return str[0:1], 1
	}
	// Scan alphanumerics.
	var i int
	for i = 0; i < len(str) && isAlphaNum(str[i]); i++ {
	}
	return str[:i], i
}

// isShellSpecialVar reports whether the character identifies a special shell
// variable such as $*.
func isShellSpecialVar(c uint8) bool {
	switch c {
	case '*', '#', '$', '@', '!', '?', '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		return true
	}
	return false
}

// isAlphaNum reports whether the byte is an ASCII letter, number, or
// underscore.
func isAlphaNum(c uint8) bool {
	return c == '_' || '0' <= c && c <= '9' || 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func OSExpandEnvMap(env, existingEnv map[string]string, projectDir string) map[string]string {
	var mapperfunc func(value string) string
	mapperfunc = func(value string) string {
		name, def, hasDefault := SplitDefault(value)

		// Special variables that should return correct value
		var val string
		if name == "PWD" {
			val = projectDir
		} else if existingEnv != nil {
			val = existingEnv[name]
		}

		if val == "" && hasDefault {
			// Allow the default to reference other variables,
			// e.g. ${FOO:-$BAR}.
			return Expand(def, mapperfunc)
		}
		return val
	}

	res := map[string]string{}
	for k, v := range env {
		res[k] = Expand(v, mapperfunc)
	}
	return res
}
