// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package writer

import "io"

type DevboxIOWriter struct {
	W     io.Writer
	Quiet bool
}

func (d DevboxIOWriter) Write(p []byte) (int, error) {
	if !d.Quiet {
		n, err := d.W.Write(p)
		if err != nil {
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}
