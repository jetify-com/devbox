// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package writer

import (
	"io"

	"github.com/spf13/cobra"
)

type devboxIOWriter struct {
	w     io.Writer
	quiet bool
}

func (d devboxIOWriter) Write(p []byte) (int, error) {
	if !d.quiet {
		n, err := d.w.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

func New(cmd *cobra.Command) *devboxIOWriter {
	quiet, err := cmd.Flags().GetBool("quiet")
	if err != nil {
		// default value for quiet/q flag
		quiet = false
	}
	return &devboxIOWriter{w: cmd.ErrOrStderr(), quiet: quiet}
}
