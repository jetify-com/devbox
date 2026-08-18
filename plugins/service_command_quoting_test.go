package plugins

import (
	"io/fs"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestServiceCommandTemplatePathsAreQuoted guards against a regression where a
// builtin plugin's process-compose service command references a templated
// filesystem path (for example "{{ .DevboxDir }}/php-fpm.conf") without
// wrapping it in double quotes. Because those templates expand to the
// project's absolute path, an unquoted reference is word-split by the shell
// whenever the project directory contains a space, and the service fails to
// start. See jetify-com/devbox#2631.
func TestServiceCommandTemplatePathsAreQuoted(t *testing.T) {
	files := processComposeFiles(t)

	for _, file := range files {
		t.Run(file, func(t *testing.T) {
			for _, command := range serviceCommands(t, file) {
				quoted := shellQuotedPositions(command)
				for _, idx := range templateIndices(command) {
					if !quoted[idx] {
						t.Errorf(
							"%s: service command %q has an unquoted templated path; "+
								"wrap it in double quotes so it survives project paths with spaces",
							file, command,
						)
					}
				}
			}
		})
	}
}

// TestServiceCommandEnvPathsAreQuoted guards against the same regression for
// service commands that reference a path through an environment variable (for
// example "$CADDY_CONFIG", which expands to "{{ .DevboxDir }}/Caddyfile").
// Only variables that are known to hold an absolute project path are checked;
// variables such as ports do not need quoting. See jetify-com/devbox#2631.
func TestServiceCommandEnvPathsAreQuoted(t *testing.T) {
	// pathEnvVars maps a plugin's process-compose.yaml to the environment
	// variables it references that expand to an absolute project path and so
	// must be quoted wherever they appear in a service command.
	pathEnvVars := map[string][]string{
		"apache/process-compose.yaml": {
			"HTTPD_CONFDIR",
			"HTTPD_ERROR_LOG_FILE",
			"HTTPD_ACCESS_LOG_FILE",
		},
		"caddy/process-compose.yaml":      {"CADDY_CONFIG"},
		"postgresql/process-compose.yaml": {"PGHOST"},
		"redis/process-compose.yaml":      {"REDIS_CONF"},
		"valkey/process-compose.yaml":     {"VALKEY_CONF"},
	}

	for file, vars := range pathEnvVars {
		t.Run(file, func(t *testing.T) {
			commands := serviceCommands(t, file)
			for _, command := range commands {
				quoted := shellQuotedPositions(command)
				for _, name := range vars {
					for _, idx := range envVarIndices(command, name) {
						if !quoted[idx] {
							t.Errorf(
								"%s: service command %q references $%s unquoted; "+
									"wrap it in double quotes so it survives project paths with spaces",
								file, command, name,
							)
						}
					}
				}
			}
		})
	}
}

// processComposeFiles returns the embedded path of every builtin plugin's
// process-compose.yaml.
func processComposeFiles(t *testing.T) []string {
	t.Helper()
	files, err := fs.Glob(builtIn, "*/process-compose.yaml")
	if err != nil {
		t.Fatalf("globbing process-compose files: %v", err)
	}
	if len(files) == 0 {
		t.Fatal("no process-compose.yaml files found")
	}
	return files
}

// serviceCommands parses a process-compose.yaml and returns every "command"
// string it defines (start, shutdown, and readiness-probe commands alike).
func serviceCommands(t *testing.T, file string) []string {
	t.Helper()
	contents, err := fs.ReadFile(builtIn, file)
	if err != nil {
		t.Fatalf("reading %s: %v", file, err)
	}

	var doc any
	if err := yaml.Unmarshal(contents, &doc); err != nil {
		t.Fatalf("parsing %s: %v", file, err)
	}

	var commands []string
	collectCommands(doc, &commands)
	return commands
}

// collectCommands walks a decoded YAML document and appends the string value
// of every mapping entry whose key is "command".
func collectCommands(node any, out *[]string) {
	switch val := node.(type) {
	case map[string]any:
		for key, child := range val {
			if key == "command" {
				if s, ok := child.(string); ok {
					*out = append(*out, s)
					continue
				}
			}
			collectCommands(child, out)
		}
	case map[any]any:
		for key, child := range val {
			if key == "command" {
				if s, ok := child.(string); ok {
					*out = append(*out, s)
					continue
				}
			}
			collectCommands(child, out)
		}
	case []any:
		for _, child := range val {
			collectCommands(child, out)
		}
	}
}

// envVarIndices returns the index of the leading '$' for every reference to the
// named environment variable in line, matching both the "$NAME" and "${NAME}"
// forms. The "$NAME" form only matches when NAME is not immediately followed by
// another identifier character, so "$FOO" does not match a search for "$F".
func envVarIndices(line, name string) []int {
	var indices []int
	for _, pattern := range []string{"${" + name + "}", "$" + name} {
		for start := 0; ; {
			i := strings.Index(line[start:], pattern)
			if i < 0 {
				break
			}
			at := start + i
			end := at + len(pattern)
			// For the bare "$NAME" form, reject a longer identifier match.
			if !strings.HasSuffix(pattern, "}") && end < len(line) && isIdentByte(line[end]) {
				start = at + 1
				continue
			}
			indices = append(indices, at)
			start = end
		}
	}
	return indices
}

func isIdentByte(b byte) bool {
	return b == '_' ||
		(b >= '0' && b <= '9') ||
		(b >= 'a' && b <= 'z') ||
		(b >= 'A' && b <= 'Z')
}
