// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"io"
	"os"

	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/ux"
)

func Clean(dir string, filesToDelete []string, w io.Writer) error {
	for _, f := range filesToDelete {
		if fileutil.Exists(f) {
			ux.Finfof(w, "Deleting %s\n", f)
		}
		if err := os.RemoveAll(dir + f); err != nil {
			return err
		}
	}

	// TODO: should the devbox shell be killed here?

	return nil
}
