package nix

import (
	"context"
	"encoding/json"
	"time"

	"go.jetify.com/devbox/nix/flake"
	"go.jetify.com/pkg/filecache"
)

const flakeCacheTTL = time.Hour * 24 * 30

var flakeFileCache = filecache.New[FlakeMetadata]("devbox/flakes")

type FlakeMetadata struct {
	Description  string    `json:"description"`
	LastModified int64     `json:"lastModified"`
	Locked       flake.Ref `json:"locked"`
	Original     flake.Ref `json:"original"`
	Path         string    `json:"path"`
	Resolved     flake.Ref `json:"resolved"`
}

// ResolveFlake runs `nix flake metadata` for the given ref. When refresh is
// true, `--refresh` is passed so nix bypasses its own eval/tarball cache and
// re-queries the upstream (e.g. GitHub) — use this on `devbox update`, not on
// paths like Add where stale-but-cached results are fine.
func ResolveFlake(ctx context.Context, ref flake.Ref, refresh bool) (FlakeMetadata, error) {
	args := []any{"flake", "metadata", "--json"}
	if refresh {
		args = append(args, "--refresh")
	}
	args = append(args, ref)
	cmd := Command(args...)
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
		meta, err := ResolveFlake(ctx, ref, false)
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
