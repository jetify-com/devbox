// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import "context"

// These make it easier to stub out nix for testing
type Nix struct{}

type Nixer interface {
	PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*PrintDevEnvOut, error)
}
