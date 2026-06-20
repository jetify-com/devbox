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
				for _, idx := range templateIndices(line) {
					if !insideDoubleQuotes(line, idx) {
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

// insideDoubleQuotes reports whether the byte at pos is enclosed in an
// unescaped pair of double quotes.
func insideDoubleQuotes(line string, pos int) bool {
	inQuotes := false
	for i := 0; i < pos && i < len(line); i++ {
		if line[i] == '"' && (i == 0 || line[i-1] != '\\') {
			inQuotes = !inQuotes
		}
	}
	return inQuotes
}
