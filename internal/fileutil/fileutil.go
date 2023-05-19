// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
	"os"
	"strings"
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

// FileContains checks if a given file at 'path' contains the 'substring'
func FileContains(path string, substring string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), substring), nil
}
