package devbox

import (
	"context"

	"go.jetpack.io/devbox/internal/nix"
)

func (d *Devbox) CacheCopy(ctx context.Context, cacheURL string) error {
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	return nix.CopyInstallableToCache(ctx, d.stderr, cacheURL, profilePath)
}
