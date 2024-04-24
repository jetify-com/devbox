package identity

import (
	"context"
	"os"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/pkg/api"
	"go.jetpack.io/pkg/auth"
	"go.jetpack.io/pkg/auth/session"
	"go.jetpack.io/pkg/ids"
	"go.jetpack.io/typeid"
	"golang.org/x/oauth2"
)

var scopes = []string{"openid", "offline_access", "email", "profile"}

type Provider struct{}

var singleton *Provider = &Provider{}

func Get() *Provider {
	return singleton
}

func (p *Provider) GenSession(ctx context.Context) (*session.Token, error) {
	if t, err := p.getTokenFromPAT(ctx); err != nil || t != nil {
		return t, err
	}

	c, err := p.AuthClient()
	if err != nil {
		return nil, err
	}
	return c.GetSession(ctx)
}

func (p *Provider) AuthClient() (*auth.Client, error) {
	return auth.NewClient(
		build.Issuer(),
		build.ClientID(),
		scopes,
		build.SuccessRedirect(),
		build.Audience(),
	)
}

func (p *Provider) getTokenFromPAT(ctx context.Context) (*session.Token, error) {
	apiKey := os.Getenv("DEVBOX_API_KEY")
	if apiKey == "" {
		return nil, nil
	}

	patID, err := typeid.Parse[ids.APIKey](apiKey)
	if err != nil {
		return nil, err
	}

	apiClient := api.NewClient(ctx, build.JetpackAPIHost(), &session.Token{})
	response, err := apiClient.GetAccessToken(ctx, patID)
	if err != nil {
		return nil, err
	}

	// This is not the greatest. This token is missing id, refresh, etc.
	// It may be better to change api.NewClient() to take a token string instead.
	return &session.Token{
		Token: oauth2.Token{
			AccessToken: response.AccessToken,
		},
	}, nil
}
