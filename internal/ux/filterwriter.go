package ux

import (
	"bytes"
	"io"
)

type filterWriter struct {
	w        io.Writer
	filtered []byte
}

func (fw *filterWriter) Write(p []byte) (n int, err error) {
	if bytes.Contains(p, fw.filtered) {
		return len(p), nil
	}
	return fw.w.Write(p)
}

// NewFilterWriter returns a writer that filters out all writes that contain the
// given string.
func NewFilterWriter(w io.Writer, f string) io.Writer {
	return &filterWriter{w: w, filtered: []byte(f)}
}
