package autodetect

import (
	"context"
	"fmt"
	"io"

	"go.jetpack.io/devbox/internal/autodetect/detector"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

func PopulateConfig(ctx context.Context, path string, stderr io.Writer) error {
	pkgs, err := packages(ctx, path)
	if err != nil {
		return err
	}
	devbox, err := devbox.Open(&devopt.Opts{
		Dir:    path,
		Stderr: stderr,
	})
	if err != nil {
		return err
	}
	return devbox.Add(ctx, pkgs, devopt.AddOpts{})
}

func DryRun(ctx context.Context, path string, stderr io.Writer) error {
	pkgs, err := packages(ctx, path)
	if err != nil {
		return err
	} else if len(pkgs) == 0 {
		fmt.Fprintln(stderr, "No packages to add")
		return nil
	}
	fmt.Fprintln(stderr, "Packages to add:")
	for _, pkg := range pkgs {
		fmt.Fprintln(stderr, pkg)
	}
	return nil
}

func detectors(path string) []detector.Detector {
	return []detector.Detector{
		&detector.PythonDetector{Root: path},
		&detector.PoetryDetector{Root: path},
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
		score, err := detector.IsRelevant(path)
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
