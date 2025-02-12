package detector

import (
	"context"
	"os"
	"path/filepath"
)

type PythonDetector struct {
	Root string
}

var _ Detector = &PythonDetector{}

func (d *PythonDetector) Relevance(path string) (float64, error) {
	requirementsPath := filepath.Join(d.Root, "requirements.txt")
	_, err := os.Stat(requirementsPath)
	if err == nil {
		return d.maxRelevance(), nil
	}
	if os.IsNotExist(err) {
		return 0, nil
	}
	return 0, err
}

func (d *PythonDetector) Packages(ctx context.Context) ([]string, error) {
	return []string{"python@latest"}, nil
}

func (d *PythonDetector) Env(ctx context.Context) (map[string]string, error) {
	return map[string]string{}, nil
}

func (d *PythonDetector) maxRelevance() float64 {
	return 1.0
}
