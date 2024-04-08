package devbox

import (
	"context"
	"io"

	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	"go.jetpack.io/devbox/internal/devbox/providers/nixcache"
	"go.jetpack.io/devbox/internal/nix"
)

func (d *Devbox) UploadProjectToCache(
	ctx context.Context,
	cacheURI string,
) error {
	var err error
	var creds types.Credentials
	if cacheURI == "" {
		cacheURI, err = d.providers.NixCache.URI(ctx)
		if err != nil {
			return err
		}
		creds, err = d.providers.NixCache.Credentials(ctx)
		if err != nil {
			return err
		}
	}
	profilePath, err := d.profilePath()
	if err != nil {
		return err
	}

	return nix.CopyInstallableToCache(
		ctx,
		d.stderr, cacheURI, profilePath, awsCredentialsToEnvVars(creds))
}

func UploadInstallableToCache(
	ctx context.Context,
	stderr io.Writer,
	cacheURI, installable string,
) error {
	var err error
	var creds types.Credentials
	if cacheURI == "" {
		cacheURI, err = nixcache.Get().URI(ctx)
		if err != nil {
			return err
		}
		creds, err = nixcache.Get().Credentials(ctx)
		if err != nil {
			return err
		}
	}
	return nix.CopyInstallableToCache(
		ctx,
		stderr, cacheURI, installable, awsCredentialsToEnvVars(creds))
}

func awsCredentialsToEnvVars(creds types.Credentials) []string {
	env := []string{}
	if creds.AccessKeyId != nil {
		env = append(env, "AWS_ACCESS_KEY_ID="+*creds.AccessKeyId)
	}
	if creds.SecretKey != nil {
		env = append(env, "AWS_SECRET_ACCESS_KEY="+*creds.SecretKey)
	}
	if creds.SessionToken != nil {
		env = append(env, "AWS_SESSION_TOKEN="+*creds.SessionToken)
	}
	return env
}
