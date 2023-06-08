// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package auth

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
)

// Authenticator performs various auth0 login flows to authenticate users.
type Authenticator struct {
	ClientID    string
	Domain      string
	Scope       string
	Audience    string
	OpenBrowser bool
	writer      io.Writer
}

// NewAuthenticator creates an authenticator that uses the auth0 production
// tenancy.
func NewAuthenticator(writer io.Writer) *Authenticator {
	return &Authenticator{
		ClientID:    "5PusB4fMm6BQ8WbTFObkTI0JUDi9ahPC",
		Domain:      "auth.jetpack.io",
		Scope:       "openid offline_access email profile",
		Audience:    "https://api.jetpack.io",
		OpenBrowser: true,
		writer:      writer,
	}
}

// DeviceAuthFlow starts decide auth flow
func (a *Authenticator) DeviceAuthFlow(ctx context.Context) error {
	resp, err := a.requestDeviceCode()
	if err != nil {
		return err
	}

	fmt.Fprintf(a.writer, "\nYour auth code is: %s\n\n", resp.UserCode)
	a.showVerificationURL(resp.VerificationURIComplete)

	tokenSuccess, err := a.requestTokens(ctx, resp)
	if err != nil {
		return err
	}

	if err = cuecfg.WriteFile(getAuthFilePath(), tokenSuccess); err != nil {
		return err
	}

	fmt.Fprintln(a.writer, "You are now authenticated.")
	return nil
}

// Use existing refresh tokens to cycle all tokens. This will fail if refresh
// tokens are missing or expired. Handle accordingly
func (a *Authenticator) RefreshTokens() (*tokenSet, error) {
	tokens := &tokenSet{}
	if err := cuecfg.ParseFile(getAuthFilePath(), tokens); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil,
				usererr.New("You must have previously logged in to use this command")
		}
		return nil, err
	}

	tokens, err := a.doRefreshToken(tokens.RefreshToken)
	if err != nil {
		return nil, err
	}

	return tokens, cuecfg.WriteFile(getAuthFilePath(), tokens)
}

func (a *Authenticator) Logout() error {
	return os.Remove(getAuthFilePath())
}
