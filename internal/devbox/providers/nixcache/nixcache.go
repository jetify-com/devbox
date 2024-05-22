package nixcache

import (
	"context"
	"slices"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devbox/providers/identity"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/pkg/api"
	nixv1alpha1 "go.jetpack.io/pkg/api/gen/priv/nix/v1alpha1"
	"go.jetpack.io/pkg/auth"
	"go.jetpack.io/pkg/auth/session"
	"go.jetpack.io/pkg/filecache"
)

type provider struct{}

var singleton *provider = &provider{}

func GetProvider() *provider {
	return singleton
}

// Credentials fetches short-lived credentials that grant access to the user's
// private cache.
func (p *provider) Credentials(ctx context.Context) (AWSCredentials, error) {
	cache := filecache.New[AWSCredentials]("devbox/providers/nixcache")
	token, err := identity.GetProvider().GenSession(ctx)
	if err != nil {
		return AWSCredentials{}, err
	}
	creds, err := cache.GetOrSetWithTime(
		"credentials-"+getSubOrAccessTokenHash(token),
		func() (AWSCredentials, time.Time, error) {
			token, err := identity.GetProvider().GenSession(ctx)
			if err != nil {
				return AWSCredentials{}, time.Time{}, err
			}
			client := api.NewClient(ctx, build.JetpackAPIHost(), token)
			creds, err := client.GetAWSCredentials(ctx)
			if err != nil {
				return AWSCredentials{}, time.Time{}, err
			}
			exp := time.Time{}
			if t := creds.GetExpiration(); t != nil {
				exp = t.AsTime()
			}
			return newAWSCredentials(creds), exp, nil
		},
	)
	if err != nil {
		return AWSCredentials{}, redact.Errorf("nixcache: get credentials: %w", redact.Safe(err))
	}
	return creds, nil
}

// Caches return the list of caches the user has access to. If user is not
// logged in, it returns nil, nil. (no error).
func (p *provider) Caches(
	ctx context.Context,
) ([]*nixv1alpha1.NixBinCache, error) {
	token, err := identity.GetProvider().GenSession(ctx)
	if errors.Is(err, auth.ErrNotLoggedIn) {
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	client := api.NewClient(ctx, build.JetpackAPIHost(), token)
	resp, err := client.GetBinCache(ctx)
	if err != nil {
		return nil, redact.Errorf("nixcache: get caches: %w", redact.Safe(err))
	}
	return resp.GetCaches(), nil
}

var readCaches []*nixv1alpha1.NixBinCache = nil

func (p *provider) CachedReadCaches(
	ctx context.Context,
) ([]*nixv1alpha1.NixBinCache, error) {
	if readCaches == nil {
		caches, err := p.Caches(ctx)
		if err != nil {
			return nil, err
		}
		readCaches = lo.Filter(caches, func(c *nixv1alpha1.NixBinCache, _ int) bool {
			return slices.Contains(
				c.GetPermissions(),
				nixv1alpha1.Permission_PERMISSION_READ,
			)
		})
	}
	return readCaches, nil
}

func (p *provider) WriteCaches(
	ctx context.Context,
) ([]*nixv1alpha1.NixBinCache, error) {
	caches, err := p.Caches(ctx)
	if err != nil {
		return nil, err
	}
	return lo.Filter(caches, func(c *nixv1alpha1.NixBinCache, _ int) bool {
		return slices.Contains(
			c.GetPermissions(),
			nixv1alpha1.Permission_PERMISSION_WRITE,
		)
	}), nil
}

func (p *provider) CachedS3Client(
	ctx context.Context,
) (*s3.Client, error) {
	creds, err := p.Credentials(ctx)
	if err != nil {
		return nil, err
	}
	config, err := config.LoadDefaultConfig(
		ctx,
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				creds.AccessKeyID,
				creds.SecretAccessKey,
				creds.SessionToken,
			),
		),
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return s3.NewFromConfig(config), nil
}

// AWSCredentials are short-lived credentials that grant access to a private Nix
// cache in S3. It marshals to JSON per the schema described in
// `aws help config-vars` under "Sourcing Credentials From External Processes".
type AWSCredentials struct {
	// Version must always be 1.
	Version         int       `json:"Version"`
	AccessKeyID     string    `json:"AccessKeyId"`
	SecretAccessKey string    `json:"SecretAccessKey"`
	SessionToken    string    `json:"SessionToken"`
	Expiration      time.Time `json:"Expiration"`
}

func newAWSCredentials(proto *nixv1alpha1.AWSCredentials) AWSCredentials {
	creds := AWSCredentials{
		Version:         1,
		AccessKeyID:     proto.AccessKeyId,
		SecretAccessKey: proto.SecretKey,
		SessionToken:    proto.SessionToken,
	}
	if proto.Expiration != nil {
		creds.Expiration = proto.Expiration.AsTime()
	}
	return creds
}

// Env returns the credentials as a slice of environment variables.
func (a AWSCredentials) Env() []string {
	return []string{
		"AWS_ACCESS_KEY_ID=" + a.AccessKeyID,
		"AWS_SECRET_ACCESS_KEY=" + a.SecretAccessKey,
		"AWS_SESSION_TOKEN=" + a.SessionToken,
	}
}

func getSubOrAccessTokenHash(token *session.Token) string {
	// We need this because the token is missing IDToken when used in CICD.
	// TODO: Implement AccessToken Parsing so we can extract sub form that.
	if token.IDClaims() != nil && token.IDClaims().Subject != "" {
		return token.IDClaims().Subject
	}
	return cachehash.Bytes([]byte(token.AccessToken))
}
