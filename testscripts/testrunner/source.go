package testrunner

import (
	"strings"

	"github.com/rogpeppe/go-internal/testscript"

	"go.jetify.com/devbox/internal/envir"
)

// Sources whatever path is exported in stdout. Ignored everything else
// Usage:
// exec devbox shellenv
// source.path
func sourcePath(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 0 {
		script.Fatalf("usage: source.path")
	}
	if neg {
		script.Fatalf("source.path does not support negation")
	}
	sourcedScript := script.ReadFile("stdout")
	for _, line := range strings.Split(sourcedScript, "\n") {
		if strings.HasPrefix(line, "export PATH=") {
			path := strings.TrimPrefix(line, "export PATH=")
			path = strings.Trim(path, "\"")
			script.Setenv(envir.Path, path)
			break
		}
	}
}
