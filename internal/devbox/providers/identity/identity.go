package identity

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"

	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
	"go.jetify.com/typeid"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/pkg/api"
	"go.jetpack.io/pkg/auth"
	"go.jetpack.io/pkg/auth/session"
	"go.jetpack.io/pkg/ids"
	"golang.org/x/oauth2"
)

// Common redirect URLs for use with [AuthClient].
var (
	// AuthRedirectDefault redirects to a generic success page.
	AuthRedirectDefault = build.SuccessRedirect()

	// AuthRedirectCache redirects to the "Cache" tab in the dashboard for
	// the authenticated organization.
	AuthRedirectCache = path.Join(build.DashboardHostname(), "team", "cache")
)

var scopes = []string{"openid", "offline_access", "email", "profile"}

var cachedAccessTokenFromAPIToken *session.Token

func GenSession(ctx context.Context) (*session.Token, error) {
	if t, err := getAccessTokenFromAPIToken(ctx); err != nil || t != nil {
		return t, err
	}

	c, err := AuthClient(AuthRedirectDefault)
	if err != nil {
		return nil, err
	}
	tok, err := c.GetSession(ctx)
	if IsRefreshTokenError(err) {
		ux.Fwarningf(os.Stderr, "Your session is expired. Please login again.\n")
		return c.LoginFlow()
	}
	return tok, err
}

func Peek() (*session.Token, error) {
	if cachedAccessTokenFromAPIToken != nil {
		return cachedAccessTokenFromAPIToken, nil
	}

	c, err := AuthClient(AuthRedirectDefault)
	if err != nil {
		return nil, err
	}
	tokens, err := c.GetSessions()
	if err != nil {
		return nil, err
	}

	if len(tokens) == 0 {
		return nil, auth.ErrNotLoggedIn
	}

	return tokens[0].Peek(), nil
}

// AuthClient returns a new client that redirects to a given URL upon success.
func AuthClient(redirect string) (*auth.Client, error) {
	return auth.NewClient(
		build.Issuer(),
		build.ClientID(),
		scopes,
		redirect,
		build.Audience(),
	)
}

func getAccessTokenFromAPIToken(
	ctx context.Context,
) (*session.Token, error) {
	if cachedAccessTokenFromAPIToken == nil {
		apiTokenRaw := os.Getenv("DEVBOX_API_TOKEN")
		if apiTokenRaw == "" {
			return nil, nil
		}

		apiToken, err := typeid.Parse[ids.APIToken](apiTokenRaw)
		if err != nil {
			return nil, err
		}

		apiClient := api.NewClient(ctx, build.JetpackAPIHost(), &session.Token{})
		response, err := apiClient.GetAccessToken(ctx, apiToken)
		if err != nil {
			return nil, err
		}

		// This is not the greatest. This token is missing id, refresh, etc.
		// It may be better to change api.NewClient() to take a token string instead.
		cachedAccessTokenFromAPIToken = &session.Token{
			Token: oauth2.Token{
				AccessToken: response.AccessToken,
			},
		}
	}

	return cachedAccessTokenFromAPIToken, nil
}

func GetOrgSlug(ctx context.Context) (string, error) {
	tok, err := GenSession(ctx)
	if err != nil {
		return "", err
	}

	if tok.IDToken == "" {
		return "", errors.New("ID token is not present")
	}

	jwt, err := jwt.ParseSigned(tok.IDToken, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		return "", err
	}

	claims := map[string]any{}
	if err = jwt.UnsafeClaimsWithoutVerification(&claims); err != nil {
		return "", err
	}

	return claims["org_trusted_metadata"].(map[string]any)["slug"].(string), nil
}

// invalid_grant or invalid_request usually means the refresh token is expired, revoked, or
// malformed. this belongs in opensource/pkg/auth
func IsRefreshTokenError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "invalid_grant") ||
		strings.Contains(err.Error(), "invalid_request")
}
