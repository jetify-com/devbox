// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cmdutil"
	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/fileutil"
)

func (p *pullbox) copyToProfile(src string) error {
	srcFileInfo, err := os.Stat(src)
	if err != nil {
		return errors.WithStack(err)
	}

	var srcFiles []fs.FileInfo
	if srcFileInfo.IsDir() {
		entries, err := os.ReadDir(src)
		if err != nil {
			return errors.WithStack(err)
		}
		for _, entry := range entries {
			info, err := entry.Info()
			if err != nil {
				return errors.WithStack(err)
			}
			srcFiles = append(srcFiles, info)
		}
	} else {
		srcFiles = []fs.FileInfo{srcFileInfo}
	}

	if err := fileutil.ClearDir(p.ProjectDir()); err != nil {
		return err
	}

	for _, srcFile := range srcFiles {
		srcPath := src
		if srcFileInfo.IsDir() {
			srcPath = filepath.Join(src, srcFile.Name())
		}
		cmd := cmdutil.CommandTTY("cp", "-rf", srcPath, p.ProjectDir())
		if err := cmd.Run(); err != nil {
			return err
		}
	}
	return nil
}

func profileIsNotEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, errors.WithStack(err)
	}
	for _, entry := range entries {
		if entry.Name() != configfile.DefaultName ||
			isModifiedConfig(filepath.Join(path, entry.Name())) {
			return true, nil
		}
	}
	return false, nil
}

func isModifiedConfig(path string) bool {
	if filepath.Base(path) == configfile.DefaultName {
		return !devconfig.IsDefault(path)
	}
	return false
}

// urlIsArchive checks if a file URL points to an archive file
func urlIsArchive(url string) (bool, error) {
	response, err := http.Head(url)
	if err != nil {
		return false, err
	}
	defer response.Body.Close()
	contentType := response.Header.Get("Content-Type")
	return strings.Contains(contentType, "tar") ||
		strings.Contains(contentType, "zip") ||
		strings.Contains(contentType, "octet-stream"), nil
}
