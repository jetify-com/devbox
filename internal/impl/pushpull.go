// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"

	"go.jetpack.io/devbox/internal/pullbox"
)

func (d *Devbox) Pull(ctx context.Context, overwrite bool, path string) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", path)
	return pullbox.New(d, path, overwrite).Pull()
}

func (d *Devbox) Push(ctx context.Context, overwrite bool, url string) error {
	fmt.Fprintf(d.writer, "Pushing global config to %s\n", url)
	return pullbox.New(d, url, overwrite).Push()
}
