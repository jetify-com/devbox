package plugins

import (
	"encoding/json"
	"io/fs"
	"strings"
	"testing"
)

// TestInitHookPathsAreQuoted guards against a regression where a builtin
// plugin's init_hook references a templated filesystem path (for example
// "{{ .Virtenv }}/bin/venvShellHook.sh") without wrapping it in double
// quotes. Because .Virtenv (and similar templates) expand to the absolute
// project path, an unquoted reference breaks the shell hook whenever the
// project directory contains a space. See jetify-com/devbox#2673.
func TestInitHookPathsAreQuoted(t *testing.T) {
	entries, err := Builtins()
	if err != nil {
		t.Fatalf("listing builtin plugins: %v", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".json") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			contents, err := fs.ReadFile(builtIn, name)
			if err != nil {
				t.Fatalf("reading %s: %v", name, err)
			}

			var plugin struct {
				Shell struct {
					InitHook json.RawMessage `json:"init_hook"`
				} `json:"shell"`
			}
			if err := json.Unmarshal(contents, &plugin); err != nil {
				t.Fatalf("parsing %s: %v", name, err)
			}

			for _, line := range initHookLines(t, plugin.Shell.InitHook) {
				quoted := shellQuotedPositions(line)
				for _, idx := range templateIndices(line) {
					if !quoted[idx] {
						t.Errorf(
							"%s: init_hook line %q has an unquoted templated path; "+
								"wrap it in double quotes so it survives project paths with spaces",
							name, line,
						)
					}
				}
			}
		})
	}
}

// initHookLines normalizes the init_hook field, which may be either a single
// string or an array of strings, into a slice of strings.
func initHookLines(t *testing.T, raw json.RawMessage) []string {
	t.Helper()
	if len(raw) == 0 {
		return nil
	}

	var list []string
	if err := json.Unmarshal(raw, &list); err == nil {
		return list
	}

	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		return []string{single}
	}

	t.Fatalf("init_hook is neither a string nor an array of strings: %s", raw)
	return nil
}

// templateIndices returns the starting index of every "{{" template opener in
// the line.
func templateIndices(line string) []int {
	var indices []int
	for offset := 0; ; {
		i := strings.Index(line[offset:], "{{")
		if i < 0 {
			return indices
		}
		indices = append(indices, offset+i)
		offset += i + 2
	}
}

// shellQuotedPositions returns a slice the same length as line where element i
// reports whether the byte at index i lies inside a properly closed shell quote
// (single or double). It models the parts of POSIX shell quoting that matter for
// word-splitting protection:
//
//   - Single and double quotes are mutually exclusive: a quote character is
//     literal (does not open a region) while inside the other quote type.
//   - A backslash escapes a double quote inside a double-quoted region.
//   - A quote that is never closed leaves its bytes unquoted, so an
//     unterminated segment is reported as unsafe rather than safe.
func shellQuotedPositions(line string) []bool {
	quoted := make([]bool, len(line))
	const none = byte(0)
	open := none // the quote char of the region currently open, or none
	start := -1  // index of the opening quote of the current region
	for i := 0; i < len(line); i++ {
		char := line[i]
		switch open {
		case none:
			if char == '\'' || char == '"' {
				open = char
				start = i
			}
		case '\'':
			// Inside single quotes nothing is special except the closing '.
			if char == '\'' {
				markQuoted(quoted, start+1, i)
				open, start = none, -1
			}
		case '"':
			if char == '"' && line[i-1] != '\\' {
				markQuoted(quoted, start+1, i)
				open, start = none, -1
			}
		}
	}
	// A region left open at end-of-line was never closed: its bytes stay unquoted.
	return quoted
}

func markQuoted(quoted []bool, lo, hi int) {
	for i := lo; i < hi; i++ {
		quoted[i] = true
	}
}
