package multi

import (
	"io/fs"
	"path/filepath"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
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

			if !dirEntry.IsDir() && filepath.Base(path) == configfile.DefaultName {
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
