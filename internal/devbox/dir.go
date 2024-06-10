// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/fileutil"
)

// findProjectDir walks up the directory tree looking for a devbox.json
// and upon finding it, will return the directory-path.
//
// If it doesn't find any devbox.json, then an error is returned.
func findProjectDir(path string) (string, error) {
	slog.Debug("finding devbox config", "path", path)

	// Sanitize the directory and use the absolute path as canonical form
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", errors.WithStack(err)
	}

	// If the path  is specified, then we check directly for a config.
	// Otherwise, we search the parent directories.
	if path != "" {
		return findProjectDirAtPath(absPath)
	}
	return findProjectDirFromParentDirSearch("/" /*root*/, absPath)
}

func findProjectDirAtPath(absPath string) (string, error) {
	fi, err := os.Stat(absPath)
	if err != nil {
		return "", err
	}

	switch mode := fi.Mode(); {
	case mode.IsDir():
		if !configExistsIn(absPath) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		return absPath, nil
	default: // assumes 'file' i.e. mode.IsRegular()
		if !fileutil.Exists(filepath.Clean(absPath)) {
			return "", missingConfigError(absPath, false /*didCheckParents*/)
		}
		// we return a directory from this function
		return filepath.Dir(absPath), nil
	}
}

func findProjectDirFromParentDirSearch(
	root string,
	absPath string,
) (string, error) {
	cur := absPath
	// Search parent directories for a devbox.json
	for cur != root {
		slog.Debug("finding devbox config", "dir", cur)
		if configExistsIn(cur) {
			return cur, nil
		}
		cur = filepath.Dir(cur)
	}
	if configExistsIn(cur) {
		return cur, nil
	}
	return "", missingConfigError(absPath, true /*didCheckParents*/)
}

func missingConfigError(path string, didCheckParents bool) error {
	var workingDir string
	wd, err := os.Getwd()
	if err == nil {
		workingDir = wd
	}
	// We try to prettify the `path` before printing
	if path == "." || path == "" || workingDir == path {
		path = "this directory"
	} else {
		// Instead of a long absolute directory, print the relative directory

		// if an error occurs, then just use `path`
		if workingDir != "" {
			relDir, err := filepath.Rel(workingDir, path)
			if err == nil {
				path = relDir
			}
		}
	}

	parentDirCheckAddendum := ""
	if didCheckParents {
		parentDirCheckAddendum = ", or any parent directories"
	}

	return usererr.New(
		"No devbox.json found in %s%s. Did you run `devbox init` yet?",
		path,
		parentDirCheckAddendum,
	)
}

func configExistsIn(path string) bool {
	return fileutil.Exists(filepath.Join(path, configfile.DefaultName))
}
