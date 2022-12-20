package golang

import (
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"golang.org/x/mod/modfile"
)

var versionMap = map[string]string{
	// Map go versions to the corresponding nixpkgs:
	"1.19": "go_1_19",
	"1.18": "go",
	"1.17": "go_1_17",
}

const defaultPkg = "go_1_19" // Default to "latest" for cases where we can't determine a version.

type Suggestor struct{}

// implements interface Suggestor (compile-time check)
var _ suggestors.Suggestor = (*Suggestor)(nil)

func (p *Suggestor) IsRelevant(srcDir string) bool {
	goModPath := filepath.Join(srcDir, "go.mod")
	return fileExists(goModPath)
}

func (p *Suggestor) Packages(srcDir string) []string {
	goPkg := getGoPackage(srcDir)

	return []string{goPkg}
}

func getGoPackage(srcDir string) string {
	goModPath := filepath.Join(srcDir, "go.mod")
	goVersion := parseGoVersion(goModPath)
	v, ok := versionMap[goVersion]
	if ok {
		return v
	} else {
		// Should we be throwing an error instead, if we don't have a nix package
		// for the specified version of go?
		return defaultPkg
	}
}

func parseGoVersion(gomodPath string) string {
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		return ""
	}
	parsed, err := modfile.ParseLax(gomodPath, content, nil)
	if err != nil {
		return ""
	}
	if parsed.Go == nil {
		return ""
	}
	return parsed.Go.Version
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
