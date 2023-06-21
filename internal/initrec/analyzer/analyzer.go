// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package analyzer

import (
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

// Analyzer helps understand the source code present in a given directory
// Handy when implementing new Planners that need to analyze files in order
// to determine what to do.
type Analyzer struct {
	rootDir string
}

func NewAnalyzer(rootDir string) (*Analyzer, error) {
	abs, err := filepath.Abs(rootDir)
	if err != nil {
		return nil, err
	}

	return &Analyzer{
		rootDir: abs,
	}, nil
}

// AbsPath resolves the given path and turns it into an absolute path relative
// to the root directory of the analyzer. If the given path is already absolute
// it leaves it as is.
func (a *Analyzer) AbsPath(path string) string {
	if filepath.IsAbs(path) {
		return path
	}

	return filepath.Join(a.rootDir, path)
}

// GlobFiles returns all the files matching the given glob patterns.
// Patterns can be relative to the analyzer's root directory. Glob patterns
// support "double star" matches.
func (a *Analyzer) GlobFiles(patterns ...string) []string {
	results := []string{}

	for _, p := range patterns {
		pattern := a.AbsPath(p)
		matches, err := doublestar.FilepathGlob(pattern)
		if err != nil {
			continue
		}
		results = append(results, matches...)
	}
	return results
}

func (a *Analyzer) HasAnyFile(patterns ...string) bool {
	matches := a.GlobFiles(patterns...)
	return len(matches) > 0
}
