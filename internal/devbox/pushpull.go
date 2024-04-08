// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"runtime/trace"

	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/pullbox"
)

func (d *Devbox) Pull(ctx context.Context, opts devopt.PullboxOpts) error {
	ctx, task := trace.NewTask(ctx, "devboxPull")
	defer task.End()
	return pullbox.New(d, opts).Pull(ctx)
}

func (d *Devbox) Push(ctx context.Context, opts devopt.PullboxOpts) error {
	ctx, task := trace.NewTask(ctx, "devboxPush")
	defer task.End()
	return pullbox.New(d, opts).Push(ctx)
}
