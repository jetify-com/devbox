package ux

import (
	"bytes"
	"io"

	"github.com/samber/lo"
)

type filterWriter struct {
	w        io.Writer
	filtered [][]byte
}

func (fw *filterWriter) Write(p []byte) (n int, err error) {
	for _, filter := range fw.filtered {
		if bytes.Contains(p, filter) {
			return len(p), nil
		}
	}
	return fw.w.Write(p)
}

// NewFilterWriter returns a writer that filters out all writes that contain the
// given string(s).
func NewFilterWriter(w io.Writer, f ...string) io.Writer {
	return &filterWriter{
		w:        w,
		filtered: lo.Map(f, func(s string, _ int) []byte { return []byte(s) }),
	}
}
