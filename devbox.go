// Package devbox creates and configures Devbox development environments.
package devbox

import (
	"context"
	"io"

	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/devbox/devopt"
)

// Devbox is a Devbox development environment.
type Devbox struct {
	dx *devbox.Devbox
}

// Open loads a Devbox environment from a config file or directory.
func Open(path string) (*Devbox, error) {
	dx, err := devbox.Open(&devopt.Opts{
		Dir:    path,
		Stderr: io.Discard,
	})
	if err != nil {
		return nil, err
	}
	return &Devbox{dx: dx}, nil
}

// Install downloads and installs missing packages.
func (d *Devbox) Install(ctx context.Context) error {
	return d.dx.Install(ctx)
}
