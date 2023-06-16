// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"
	"runtime/trace"

	"go.jetpack.io/devbox/internal/pullbox"
)

func (d *Devbox) Pull(ctx context.Context, force bool, path string) error {
	ctx, task := trace.NewTask(ctx, "devboxPull")
	defer task.End()

	fmt.Fprintf(d.writer, "Pulling global config from %s\n", path)
	return pullbox.New(d, path, force).Pull(ctx)
}

func (d *Devbox) Push(ctx context.Context, url string) error {
	ctx, task := trace.NewTask(ctx, "devboxPush")
	defer task.End()

	fmt.Fprintf(d.writer, "Pushing global config to %s\n", url)
	return pullbox.New(d, url, false).Push(ctx)
}
