// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pullbox

import (
	"net/http"
	"strings"
)

type pullbox struct {
}

func New() *pullbox {
	return &pullbox{}
}

func (p *pullbox) DownloadAndExtract(overwrite bool, url, target string) error {
	data, err := download(url)
	if err != nil {
		return err
	}
	tmpDir, err := extract(data)
	if err != nil {
		return err
	}

	return p.copy(overwrite, tmpDir, target)
}

// URLIsArchive checks if a file URL points to an archive file
func (p *pullbox) URLIsArchive(url string) (bool, error) {
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
