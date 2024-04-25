package devbox

import (
	"context"
	"io"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/devbox/providers/nixcache"
	"go.jetpack.io/devbox/internal/nix"
)

func (d *Devbox) UploadProjectToCache(
	ctx context.Context,
	cacheURI string,
) error {
	if cacheURI == "" {
		var err error
		cacheURI, err = d.providers.NixCache.URI(ctx)
		if err != nil {
			return err
		}
		if cacheURI == "" {
			return usererr.New("Your account's organization doesn't have a Nix cache.")
		}
	}

	creds, err := d.providers.NixCache.Credentials(ctx)
	if err != nil {
		return err
	}
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	// Ensure state is up to date before uploading to cache.
	// TODO: we may be able to do this more efficiently, not sure everything needs
	// to be installed.
	if err = d.ensureStateIsUpToDate(ctx, ensure); err != nil {
		return err
	}

	return nix.CopyInstallableToCache(ctx, d.stderr, cacheURI, profilePath, creds.Env())
}

func UploadInstallableToCache(
	ctx context.Context,
	stderr io.Writer,
	cacheURI, installable string,
) error {
	if cacheURI == "" {
		var err error
		cacheURI, err = nixcache.Get().URI(ctx)
		if err != nil {
			return err
		}
		if cacheURI == "" {
			return usererr.New("Your account's organization doesn't have a Nix cache.")
		}
	}

	creds, err := nixcache.Get().Credentials(ctx)
	if err != nil {
		return err
	}
	return nix.CopyInstallableToCache(ctx, stderr, cacheURI, installable, creds.Env())
}
