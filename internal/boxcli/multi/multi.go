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
	// A single directory may contain more than one recognized config name
	// (e.g. both devbox.json and devbox.jsonc). Track the directories already
	// opened so each project is opened exactly once.
	seenDirs := map[string]bool{}
	err := filepath.WalkDir(
		".",
		func(path string, dirEntry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			if !dirEntry.IsDir() && slices.Contains(configfile.ValidNames, filepath.Base(path)) {
				dir := filepath.Dir(path)
				if seenDirs[dir] {
					return nil
				}
				seenDirs[dir] = true

				optsCopy := *opts
				// Open by directory so devconfig applies its filename
				// precedence (devbox.json wins over devbox.jsonc).
				optsCopy.Dir = dir
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
