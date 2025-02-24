package detector

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"go.jetify.com/devbox/internal/searcher"
)

type PoetryDetector struct {
	PythonDetector
	Root string
}

var _ Detector = &PoetryDetector{}

func (d *PoetryDetector) Relevance(path string) (float64, error) {
	pyprojectPath := filepath.Join(d.Root, "pyproject.toml")
	_, err := os.Stat(pyprojectPath)
	if err == nil {
		return d.maxRelevance(), nil
	}
	if os.IsNotExist(err) {
		return 0, nil
	}
	return 0, err
}

func (d *PoetryDetector) Packages(ctx context.Context) ([]string, error) {
	pyprojectPath := filepath.Join(d.Root, "pyproject.toml")
	pyproject, err := os.ReadFile(pyprojectPath)
	if err != nil {
		return nil, err
	}

	var pyprojectToml struct {
		Tool struct {
			Poetry struct {
				Version      string `toml:"version"`
				Dependencies struct {
					Python string `toml:"python"`
				} `toml:"dependencies"`
			} `toml:"poetry"`
		} `toml:"tool"`
	}
	err = toml.Unmarshal(pyproject, &pyprojectToml)
	if err != nil {
		return nil, err
	}

	poetryVersion := determineBestVersion(ctx, "poetry", pyprojectToml.Tool.Poetry.Version)
	pythonVersion := determineBestVersion(ctx, "python", pyprojectToml.Tool.Poetry.Dependencies.Python)

	return []string{"python@" + pythonVersion, "poetry@" + poetryVersion}, nil
}

func (d *PoetryDetector) Env(ctx context.Context) (map[string]string, error) {
	return d.PythonDetector.Env(ctx)
}

func determineBestVersion(ctx context.Context, pkg, version string) string {
	if version == "" {
		return "latest"
	}

	version = sanitizeVersion(version)

	_, err := searcher.Client().ResolveV2(ctx, pkg, version)
	if err != nil {
		return "latest"
	}

	return version
}

func sanitizeVersion(version string) string {
	// Remove non-numeric characters and 'v' prefix
	sanitized := strings.TrimPrefix(version, "v")
	return regexp.MustCompile(`[^\d.]`).ReplaceAllString(sanitized, "")
}

func (d *PoetryDetector) maxRelevance() float64 {
	// this is arbitrary, but we want to prioritize poetry over python
	return d.PythonDetector.maxRelevance() + 1
}
