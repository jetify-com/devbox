package nixcache

import (
	"context"
	"errors"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity/types"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/devbox/providers/identity"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/pkg/api"
	nixv1alpha1 "go.jetpack.io/pkg/api/gen/priv/nix/v1alpha1"
	"go.jetpack.io/pkg/auth"
	"go.jetpack.io/pkg/filecache"
)

type Provider struct{}

var singleton *Provider = &Provider{}

type NixCacheConfig struct {
	URI         string
	Credentials types.Credentials
}

func (n NixCacheConfig) CredentialsEnvVars() []string {
	env := []string{}
	if n.Credentials.AccessKeyId != nil {
		env = append(env, "AWS_ACCESS_KEY_ID="+*n.Credentials.AccessKeyId)
	}
	if n.Credentials.SecretKey != nil {
		env = append(env, "AWS_SECRET_ACCESS_KEY="+*n.Credentials.SecretKey)
	}
	if n.Credentials.SessionToken != nil {
		env = append(env, "AWS_SESSION_TOKEN="+*n.Credentials.SessionToken)
	}
	return env
}

func Get() *Provider {
	return singleton
}

// Config returns the URI or the nix bin cache and AWS credentials if available.
// Nix calls the URI a substituter.
// A substituter is a bin cache URI that nix can use to fetch pre-built
// binaries from.
func (p *Provider) Config(ctx context.Context) (NixCacheConfig, error) {
	token, err := identity.Get().GenSession(ctx)

	if errors.Is(err, auth.ErrNotLoggedIn) {
		// DEVBOX_NIX_BINCACHE_URI seems like a friendlier name than "substituter"
		return NixCacheConfig{
			URI: os.Getenv("DEVBOX_NIX_BINCACHE_URI"),
		}, nil
	} else if err != nil {
		return NixCacheConfig{}, err
	}

	apiClient := api.NewClient(ctx, build.JetpackAPIHost(), token)
	cache := filecache.New[*nixv1alpha1.GetBinCacheResponse]("devbox/credentials")
	binCacheResponse, err := cache.GetOrSetT(
		"aws-nix-bin-cache",
		func() (*nixv1alpha1.GetBinCacheResponse, time.Time, error) {
			r, err := apiClient.GetBinCache(ctx)
			if err != nil || r.GetNixBinCacheUri() == "" {
				return nil, time.Time{}, err
			}
			return r, r.GetNixBinCacheCredentials().Expiration.AsTime(), nil
		},
	)

	if err != nil {
		return NixCacheConfig{}, err
	}

	checkIfUserCanAddSubstituter(ctx)

	return NixCacheConfig{
		URI: binCacheResponse.NixBinCacheUri,
		Credentials: types.Credentials{
			AccessKeyId:  aws.String(binCacheResponse.GetNixBinCacheCredentials().GetAccessKeyId()),
			SecretKey:    aws.String(binCacheResponse.GetNixBinCacheCredentials().GetSecretKey()),
			SessionToken: aws.String(binCacheResponse.GetNixBinCacheCredentials().GetSessionToken()),
		},
	}, nil
}

func checkIfUserCanAddSubstituter(ctx context.Context) {
	// we need to ensure that the user can actually use the extra
	// substituter. If the user did a root install, then we need to add
	// the trusted user/substituter to the nix.conf file and restart the daemon.

	// This check is not perfect, so we still try to use the substituter even if
	// it fails

	// TODOs:
	// * Also check if cache is enabled in nix.conf
	// * Test on single user install
	// * Automate making user trusted if needed
	if !nix.IsUserTrusted(ctx) {
		ux.Fwarning(
			os.Stderr,
			"In order to use a custom nix cache you must be a trusted user. Please "+
				"add your username to nix.conf (usually located at /etc/nix/nix.conf)"+
				" and restart the nix daemon.",
		)
	}
}
