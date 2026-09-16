package multi

import (
	"io/fs"
	"path/filepath"
	"slices"

	"go.jetify.com/devbox/internal/debug"
	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/devbox/devopt"
	"go.jetify.com/devbox/internal/devconfig/configfile"
)

func Open(opts *devopt.Opts) ([]*devbox.Devbox, error) {
	defer debug.FunctionTimer().End()

	var boxes []*devbox.Devbox
	err := filepath.WalkDir(
		".",
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !dirEntry.IsDir() && slices.Contains(configfile.ValidNames, filepath.Base(path)) {
				optsCopy := *opts
				optsCopy.Dir = path
				box, err := devbox.Open(&optsCopy)
				if err != nil {
					return err
				}
				boxes = append(boxes, box)
			}

			return nil
		},
	)

	return boxes, err
}
