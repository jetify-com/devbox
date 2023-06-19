// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"runtime/trace"

	"go.jetpack.io/devbox/internal/pullbox"
)

func (d *Devbox) Pull(ctx context.Context, force bool, path string) error {
	ctx, task := trace.NewTask(ctx, "devboxPull")
	defer task.End()
	return pullbox.New(d, path, force).Pull(ctx)
}

func (d *Devbox) Push(ctx context.Context, url string) error {
	ctx, task := trace.NewTask(ctx, "devboxPush")
	defer task.End()
	return pullbox.New(d, url, false).Push(ctx)
}
