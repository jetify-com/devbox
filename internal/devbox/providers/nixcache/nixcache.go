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

func Get() *Provider {
	return singleton
}

// URI returns the URI of the nix bin cache if available.
// Nix calls the URI a substituter.
// A substituter is a bin cache URI that nix can use to fetch pre-built
// binaries from.
func (p *Provider) URI(ctx context.Context) (string, error) {
	token, err := identity.Get().GenSession(ctx)

	if errors.Is(err, auth.ErrNotLoggedIn) {
		// DEVBOX_NIX_BINCACHE_URI seems like a friendlier name than "substituter"
		return os.Getenv("DEVBOX_NIX_BINCACHE_URI"), nil
	} else if err != nil {
		return "", err
	}

	apiClient := api.NewClient(ctx, build.JetpackAPIHost(), token)
	cache := filecache.New[*nixv1alpha1.GetBinCacheResponse]("devbox/providers/nixcache")
	binCacheResponse, err := cache.GetOrSet(
		"uri",
		func() (*nixv1alpha1.GetBinCacheResponse, time.Duration, error) {
			r, err := apiClient.GetBinCache(ctx)
			if err != nil || r.GetNixBinCacheUri() == "" {
				return nil, 0, err
			}
			return r, time.Hour, nil
		},
	)
	if err != nil {
		return "", err
	}

	checkIfUserCanAddSubstituter(ctx)

	return binCacheResponse.NixBinCacheUri, nil
}

func (p *Provider) Credentials(ctx context.Context) (types.Credentials, error) {
	token, err := identity.Get().GenSession(ctx)

	if errors.Is(err, auth.ErrNotLoggedIn) {
		return types.Credentials{}, nil
	} else if err != nil {
		return types.Credentials{}, err
	}

	apiClient := api.NewClient(ctx, build.JetpackAPIHost(), token)
	cache := filecache.New[*nixv1alpha1.AWSCredentials]("devbox/providers/nixcache")
	credentials, err := cache.GetOrSetWithTime(
		"aws-credentials",
		func() (*nixv1alpha1.AWSCredentials, time.Time, error) {
			r, err := apiClient.GetAWSCredentials(ctx)
			if err != nil || r.GetAccessKeyId() == "" {
				return nil, time.Time{}, err
			}
			return r, r.GetExpiration().AsTime(), nil
		},
	)
	if err != nil {
		return types.Credentials{}, err
	}

	checkIfUserCanAddSubstituter(ctx)

	return types.Credentials{
		AccessKeyId:  aws.String(credentials.GetAccessKeyId()),
		SecretKey:    aws.String(credentials.GetSecretKey()),
		SessionToken: aws.String(credentials.GetSessionToken()),
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
