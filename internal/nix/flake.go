package nix

import (
	"context"
	"encoding/json"
	"time"

	"go.jetify.com/devbox/nix/flake"
	"go.jetify.com/pkg/filecache"
)

const flakeCacheTTL = time.Hour * 24 * 90

var flakeFileCache = filecache.New[FlakeMetadata]("devbox/flakes")

type FlakeMetadata struct {
	Description  string    `json:"description"`
	LastModified int64     `json:"lastModified"`
	Locked       flake.Ref `json:"locked"`
	Original     flake.Ref `json:"original"`
	Path         string    `json:"path"`
	Resolved     flake.Ref `json:"resolved"`
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

func ResolveCachedFlake(ctx context.Context, ref flake.Ref) (FlakeMetadata, error) {
	return flakeFileCache.GetOrSet(ref.String(), func() (FlakeMetadata, time.Duration, error) {
		meta, err := ResolveFlake(ctx, ref)
		if err != nil {
			return FlakeMetadata{}, 0, err
		}
		return meta, flakeCacheTTL, nil
	})
}

func ClearFlakeCache(ref flake.Ref) error {
	// TODO: Add unset to filecache
	return flakeFileCache.Set(ref.String(), FlakeMetadata{}, -1)
}
