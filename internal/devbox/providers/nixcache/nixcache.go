package nixcache

import (
	"context"
	"errors"
	"os"
	"time"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/devbox/providers/identity"
	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/setup"
	"go.jetpack.io/pkg/api"
	nixv1alpha1 "go.jetpack.io/pkg/api/gen/priv/nix/v1alpha1"
	"go.jetpack.io/pkg/filecache"
)

type Provider struct{}

var singleton *Provider = &Provider{}

func Get() *Provider {
	return singleton
}

func (p *Provider) Configure(ctx context.Context, username string) error {
	return p.configure(ctx, username, false)
}

func (p *Provider) ConfigureReprompt(ctx context.Context, username string) error {
	return p.configure(ctx, username, true)
}

func (p *Provider) configure(ctx context.Context, username string, reprompt bool) error {
	setupTasks := []struct {
		key  string
		task setup.Task
	}{
		{"nixcache-setup-aws", &awsSetupTask{username}},
		{"nixcache-setup-nix", &nixSetupTask{username}},
	}
	if reprompt {
		for _, t := range setupTasks {
			setup.Reset(t.key)
		}
	}

	// If we're already root, then do the setup without prompting the user
	// for confirmation.
	if os.Getuid() == 0 {
		for _, t := range setupTasks {
			err := setup.Run(ctx, t.key, t.task)
			if err != nil {
				return redact.Errorf("nixcache: run setup: %v", err)
			}
		}
		return nil
	}

	// Otherwise, ask the user to confirm if it's okay to sudo.
	const sudoPrompt = "Devbox requires root to configure the Nix daemon to use your organization's Devbox cache. Allow sudo?"
	for _, t := range setupTasks {
		err := setup.ConfirmRun(ctx, t.key, t.task, sudoPrompt)
		if errors.Is(err, setup.ErrUserRefused) {
			return nil
		}
		if err != nil {
			return redact.Errorf("nixcache: run setup: %v", err)
		}
	}
	return nil
}

// Credentials fetches short-lived credentials that grant access to the user's
// private cache.
func (p *Provider) Credentials(ctx context.Context) (AWSCredentials, error) {
	cache := filecache.New[AWSCredentials]("devbox/providers/nixcache")
	token, err := identity.Get().GenSession(ctx)
	if err != nil {
		return AWSCredentials{}, err
	}
	creds, err := cache.GetOrSetWithTime(
		"credentials-"+token.IDClaims().Subject,
		func() (AWSCredentials, time.Time, error) {
			token, err := identity.Get().GenSession(ctx)
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

// URI queries the Jetify API for the URI that points to user's private cache.
// If their account doesn't have access to a cache, it returns an empty string
// and a nil error.
func (p *Provider) URI(ctx context.Context) (string, error) {
	cache := filecache.New[string]("devbox/providers/nixcache")
	token, err := identity.Get().GenSession(ctx)
	if err != nil {
		return "", err
	}
	// Landau: I think we can probably remove this cache? This endpoint is very
	// fast and we only use this for build/upload which are slow.
	uri, err := cache.GetOrSet(
		"uri-"+token.IDClaims().Subject,
		func() (string, time.Duration, error) {
			client := api.NewClient(ctx, build.JetpackAPIHost(), token)
			resp, err := client.GetBinCache(ctx)
			if err != nil {
				return "", 0, redact.Errorf("nixcache: get uri: %w", redact.Safe(err))
			}

			// Don't cache negative responses.
			if resp.GetNixBinCacheUri() == "" {
				return "", 0, nil
			}

			// TODO(gcurtis): do a better job of invalidating the URI after
			// a Nix command fails to query the cache.
			return resp.GetNixBinCacheUri(), 24 * time.Hour, nil
		},
	)
	if err != nil {
		return "", redact.Errorf("nixcache: get uri: %w", redact.Safe(err))
	}
	return uri, nil
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
