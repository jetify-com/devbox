package autodetect

import (
	"context"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/pkg/autodetect/detector"
)

func InitConfig(ctx context.Context, path string) error {
	config, err := devconfig.Init(path)
	if err != nil {
		return err
	}

	if err = populateConfig(ctx, path, config); err != nil {
		return err
	}

	return config.Root.Save()
}

func DryRun(ctx context.Context, path string) ([]byte, error) {
	config := devconfig.DefaultConfig()
	if err := populateConfig(ctx, path, config); err != nil {
		return nil, err
	}
	return config.Root.Bytes(), nil
}

func populateConfig(ctx context.Context, path string, config *devconfig.Config) error {
	pkgs, err := packages(ctx, path)
	if err != nil {
		return err
	}
	for _, pkg := range pkgs {
		config.PackageMutator().Add(pkg)
	}
	return nil
}

func detectors(path string) []detector.Detector {
	return []detector.Detector{
		&detector.PythonDetector{Root: path},
		&detector.PoetryDetector{Root: path},
		&detector.GoDetector{Root: path},
	}
}

func packages(ctx context.Context, path string) ([]string, error) {
	mostRelevantDetector, err := relevantDetector(path)
	if err != nil || mostRelevantDetector == nil {
		return nil, err
	}
	return mostRelevantDetector.Packages(ctx)
}

// relevantDetector returns the most relevant detector for the given path.
// We could modify this to return a list of detectors and their scores or
// possibly grouped detectors by category (e.g. python, server, etc.)
func relevantDetector(path string) (detector.Detector, error) {
	relevantScore := 0.0
	var mostRelevantDetector detector.Detector
	for _, detector := range detectors(path) {
		score, err := detector.Relevance(path)
		if err != nil {
			return nil, err
		}
		if score > relevantScore {
			relevantScore = score
			mostRelevantDetector = detector
		}
	}
	return mostRelevantDetector, nil
}
