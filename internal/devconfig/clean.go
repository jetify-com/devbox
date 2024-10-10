// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"os"
)

func Clean(dir string) error {
	filesToDelete := []string{
		"devbox.lock",
		".devbox",
	}
	for _, f := range filesToDelete {
		// TODO: what should we do here when an unexpected error occurs? print an error?
		_ = os.RemoveAll(dir + f)
	}

	// TODO: should the devbox shell be killed here?

	return nil
}
