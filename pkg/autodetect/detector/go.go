package detector

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
)

type GoDetector struct {
	Root string
}

var _ Detector = &GoDetector{}

func (d *GoDetector) Relevance(path string) (float64, error) {
	goModPath := filepath.Join(d.Root, "go.mod")
	_, err := os.Stat(goModPath)
	if err == nil {
		return 1.0, nil
	}
	if os.IsNotExist(err) {
		return 0, nil
	}
	return 0, err
}

func (d *GoDetector) Packages(ctx context.Context) ([]string, error) {
	goModPath := filepath.Join(d.Root, "go.mod")
	goModContent, err := os.ReadFile(goModPath)
	if err != nil {
		return nil, err
	}

	// Parse the Go version from go.mod
	goVersion := parseGoVersion(string(goModContent))
	if goVersion != "" {
		return []string{"go@" + goVersion}, nil
	}
	return []string{"go@latest"}, nil
}

func parseGoVersion(goModContent string) string {
	// Use a regular expression to find the Go version directive
	re := regexp.MustCompile(`(?m)^go\s+(\d+\.\d+(\.\d+)?)`)
	match := re.FindStringSubmatch(goModContent)

	if len(match) >= 2 {
		return match[1]
	}

	return ""
}
