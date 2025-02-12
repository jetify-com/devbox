package nix

import (
	"context"
	"encoding/json"

	"go.jetpack.io/devbox/nix/flake"
)

type FlakeMetadata struct {
	Description string    `json:"description"`
	Original    flake.Ref `json:"original"`
	Resolved    flake.Ref `json:"resolved"`
	Locked      flake.Ref `json:"locked"`
	Path        string    `json:"path"`
}

func ResolveFlake(ctx context.Context, ref flake.Ref) (FlakeMetadata, error) {
	cmd := Command("flake", "metadata", "--json", ref)
	out, err := cmd.Output(ctx)
	if err != nil {
		return FlakeMetadata{}, err
	}
	meta := FlakeMetadata{}
	err = json.Unmarshal(out, &meta)
	if err != nil {
		return FlakeMetadata{}, err
	}
	return meta, nil
}
