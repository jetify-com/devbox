package nix

import "context"

// These make it easier to stub out nix for testing
type Nix struct{}

type Nixer interface {
	PrintDevEnv(ctx context.Context, args *PrintDevEnvArgs) (*PrintDevEnvOut, error)
}
