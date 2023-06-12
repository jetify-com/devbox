package impl

import (
	"context"
	"fmt"

	"go.jetpack.io/devbox/internal/pullbox"
)

func (d *Devbox) Pull(
	ctx context.Context,
	force bool,
	path string,
) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", path)
	return pullbox.New(d, path, force).Pull(ctx)
}

func (d *Devbox) Push(ctx context.Context, url string) error {
	fmt.Fprintf(d.writer, "Pushing global config\n")
	return pullbox.New(d, url, false).Push(ctx)
}
