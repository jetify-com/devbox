// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// TODO: publish as it's own shared package that other binaries can use.

// IsDir returns true if the path exists *and* it is pointing to a directory.
//
// This function will traverse symbolic links to query information about the
// destination file.
//
// This is a convenience function that coerces errors to false. If it cannot
// read the path for any reason (including a permission error, or a broken
// symbolic link) it returns false.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile returns true if the path exists *and* it is pointing to a regular file.
//
// This function will traverse symbolic links to query information about the
// destination file.
//
// This is a convenience function that coerces errors to false. If it cannot
// read the path for any reason (including a permission error, or a broken
// symbolic link) it returns false.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func IsDirEmpty(path string) bool {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false
	}
	return len(entries) == 0
}

// FileContains checks if a given file at 'path' contains the 'substring'
func FileContains(path, substring string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), substring), nil
}

func EnsureDirExists(path string, perm fs.FileMode, chmod bool) error {
	if err := os.MkdirAll(path, perm); err != nil && !errors.Is(err, fs.ErrExist) {
		return errors.WithStack(err)
	}
	if chmod {
		if err := os.Chmod(path, perm); err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
}

func EnsureAbsolutePaths(paths []string) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.WithStack(err)
	}
	absPaths := make([]string, len(paths))
	for i, path := range paths {
		if filepath.IsAbs(path) {
			absPaths[i] = path
		} else {
			absPaths[i] = filepath.Join(wd, path)
		}
	}
	return absPaths, nil
}
