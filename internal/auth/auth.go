package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/browser"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/xdg"
)

// These are messages displayed to the user upon error condition.
const (
	accessDeniedMsg     = "\n\n ERROR: Received access_denied from the Auth server. Please try `devbox auth login` again.\n\n"
	invalidGrantMsg     = "\n\n ERROR: Received invalid_grant from the Auth server. Please try `devbox auth login` again.\n\n"
	expiredTokenMsg     = "\n\n ERROR: The token has expired. Please try `devbox auth login` again.\n\n"
	maxTriesExceededMsg = "\n\n ERROR: Unable to get successful response from the Auth server. Please try `devbox auth login` again.\n\n"
)

var (
	maxTriesExceededError = errors.New("max number of tries exceeded")
	accessDeniedError     = errors.New("access was denied")
)

type codeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
}

// used for both requestToken and refreshToken functions
type requestTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

// Authenticator performs various auth0 login flows to authenticate users.
type Authenticator struct {
	ClientID    string
	Domain      string
	Scope       string
	Audience    string
	OpenBrowser bool
}

// NewAuthenticator creates an authenticator that uses the auth0 production
// tenancy.
func NewAuthenticator() *Authenticator {
	return &Authenticator{
		ClientID:    "5PusB4fMm6BQ8WbTFObkTI0JUDi9ahPC",
		Domain:      "auth.jetpack.io",
		Scope:       "openid offline_access email profile",
		Audience:    "https://api.jetpack.io",
		OpenBrowser: true,
	}
}

// DeviceAuthFlow implements authorizing the user via the CLI "Auth0 app"
// to use certain "scopes" permissions.
// reference: https://auth0.com/docs/get-started/authentication-and-authorization-flow/call-your-api-using-the-device-authorization-flow
func (a *Authenticator) DeviceAuthFlow(ctx context.Context) error {
	resp, err := a.requestDeviceCode()
	if err != nil {
		return errors.WithStack(err)
	}

	fmt.Printf("\nYour auth code is: %s\n\n", resp.UserCode)
	a.showVerificationURL(resp.VerificationURIComplete)

	tokenSuccess, err := a.requestTokens(ctx, resp)
	if err != nil {
		return errors.WithStack(err)
	}

	if err = cuecfg.WriteFile(getAuthFilePath(), tokenSuccess); err != nil {
		return errors.Wrapf(err, "failed to save AuthTokens to config")
	}

	fmt.Println("You are now authenticated.")
	return nil
}

// showVerificationURL presents a device flow verification URL to the user,
// either by printing it to stdout or opening a web browser.
func (a *Authenticator) showVerificationURL(url string) {
	opened := false
	if a.OpenBrowser {
		err := browser.OpenURL(url)
		opened = err == nil
	}
	if opened {
		fmt.Printf("Opening your browser to complete the login. "+
			"If your browser didn't open, you can go to this URL "+
			"and confirm your code manually:\n%s\n\n", url)
		return
	}
	fmt.Printf("Please go to this URL to confirm this code and login: %s\n\n", url)
}

// requestDeviceCode requests a device code that the user can use to
// authorize the device.
func (a *Authenticator) requestDeviceCode() (*codeResponse, error) {

	reqURL := fmt.Sprintf("https://%s/oauth/device/code", a.Domain)
	payload := strings.NewReader(fmt.Sprintf(
		"client_id=%s&scope=%s&audience=%s",
		a.ClientID,
		url.QueryEscape(a.Scope),
		a.Audience,
	))

	req, err := http.NewRequest(http.MethodPost, reqURL, payload)
	if err != nil {
		bytesPayload, _ := io.ReadAll(payload)
		return nil, errors.Wrapf(
			err,
			"failed to send request to URL: %s with payload: %s",
			reqURL,
			string(bytesPayload),
		)
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to send Request")
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if res.StatusCode != http.StatusOK {
		return nil, errors.Errorf(
			"got status code: %d, with body %s",
			res.StatusCode,
			string(body),
		)
	}

	response := codeResponse{}
	return &response, json.Unmarshal(body, &response)
}

// reference: https://auth0.com/docs/flows/call-your-api-using-the-device-authorization-flow#request-tokens
func (a *Authenticator) requestTokens(
	ctx context.Context,
	codeResponse *codeResponse,
) (*tokenResponse, error) {

	timeToSleep := codeResponse.Interval
	ticker := time.NewTicker(time.Duration(timeToSleep) * time.Second)
	defer ticker.Stop()

	// numTries is a counter to guard against infinite looping.
	// In the normal course:
	//    Status Code 200 OK: we early return within loop
	//    Known Error scenarios: we continue looping and requesting Auth0 API.
	//       These are not "errors" so much as "user hasn't yet completed
	//       browser login flow"
	//    Unknown Error scenarios: we early return within loop

	for numTries := 0; numTries < 100; numTries++ {
		select {
		case <-ctx.Done():
			return nil, errors.WithStack(ctx.Err())

		case <-ticker.C:
			reqURL := fmt.Sprintf("https://%s/oauth/token", a.Domain)

			grantType := "urn:ietf:params:oauth:grant-type:device_code"
			payload := strings.NewReader(fmt.Sprintf(
				"grant_type=%s&device_code=%s&client_id=%s",
				url.QueryEscape(grantType),
				codeResponse.DeviceCode,
				a.ClientID,
			))

			req, _ := http.NewRequest(http.MethodPost, reqURL, payload)

			req.Header.Add("content-type", "application/x-www-form-urlencoded")

			res, err := http.DefaultClient.Do(req)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to complete request to: %s", reqURL)
			}

			defer res.Body.Close()
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, errors.Wrap(err, "failed to read response body")
			}

			// Success Scenario
			if res.StatusCode == http.StatusOK {
				tokens := tokenResponse{}
				fmt.Println(string(body))
				return &tokens, json.Unmarshal(body, &tokens)
			}

			// Handle failure scenarios
			tokenErrorBody := requestTokenError{}
			if err := json.Unmarshal(body, &tokenErrorBody); err != nil {
				return nil, errors.Wrapf(
					err,
					"unable to unmarshal requestTokenError from body %s",
					body,
				)
			}

			if res.StatusCode == http.StatusTooManyRequests {
				if tokenErrorBody.Error != "slow_down" {
					return nil, errors.Errorf(
						"got status code: %d, response body: %s",
						res.StatusCode,
						body,
					)
				}

				// We are polling too fast. We slow down a bit.
				timeToSleep += 1
				ticker.Reset(time.Duration(timeToSleep) * time.Second)

			} else if res.StatusCode == http.StatusForbidden {

				// this error is received when waiting for user to take action
				// when they are logging in via browser. Continue polling.
				if tokenErrorBody.Error != "authorization_pending" {
					return nil, errors.Errorf(
						"got status code: %d, response body: %s",
						res.StatusCode,
						body,
					)
				}
			} else {
				// The user has not authorized the device quickly enough, so
				// the `device_code` has expired. Notify the user that the
				// flow has expired and prompt them to re-initiate the flow.
				if tokenErrorBody.Error == "expired_token" {
					fmt.Print(expiredTokenMsg)
					return nil, errors.Errorf(
						"got status code: %d, response body: %s",
						res.StatusCode,
						body,
					)
				}

				// The "expired_token" is returned exactly once. After that,
				// the dreaded "invalid_grant" will be returned and device
				// must stop polling.
				if tokenErrorBody.Error == "invalid_grant" {
					fmt.Print(invalidGrantMsg)
					return nil, errors.Errorf(
						"got status code: %d, response body: %s",
						res.StatusCode,
						body,
					)
				}

				// "access_denied" can be received for:
				// 1. user refused to authorize the device.
				// 2. Auth server denied the transaction.
				// 3. A configured Auth0 "rule" denied access
				if tokenErrorBody.Error == "access_denied" {
					fmt.Print(accessDeniedMsg)
					return nil, accessDeniedError
				}
			}
		} // end select
	} // end for

	fmt.Print(maxTriesExceededMsg)
	return nil, maxTriesExceededError
}

func (a *Authenticator) RefreshTokens() error {
	token := tokenResponse{}
	if err := cuecfg.ParseFile(getAuthFilePath(), &token); err != nil {
		return errors.WithStack(err)
	}

	tokenSuccess, err := a.doRefreshToken(token.RefreshToken)
	if err != nil {
		return errors.WithStack(err)
	}

	if err = cuecfg.WriteFile(getAuthFilePath(), tokenSuccess); err != nil {
		return errors.Wrapf(err, "failed to save AuthTokens to config")
	}

	if err != nil {
		return errors.Wrap(err, "failed writing auth tokens to Local Config")
	}

	return nil
}

func (a *Authenticator) doRefreshToken(
	refreshToken string,
) (*tokenResponse, error) {

	reqURL := fmt.Sprintf("https://%s/oauth/token", a.Domain)

	payload := fmt.Sprintf(
		"grant_type=refresh_token&client_id=%s&refresh_token=%s",
		a.ClientID,
		refreshToken,
	)
	payloadReader := strings.NewReader(payload)

	req, err := http.NewRequest(http.MethodPost, reqURL, payloadReader)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed to create request to URL: %s, with payload: %s",
			reqURL,
			payload,
		)
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrapf(
			err,
			"failed POST request to reqURL: %s, payload: %s ",
			reqURL,
			payload,
		)
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	if res.StatusCode == http.StatusOK {
		tokens := &tokenResponse{}
		return tokens, json.Unmarshal(body, tokens)
	}

	tokenErrorBody := requestTokenError{}
	if err := json.Unmarshal(body, &tokenErrorBody); err != nil {
		return nil, errors.Wrapf(
			err,
			"unable to unmarshal requestTokenError from body %s",
			body,
		)
	}
	return nil, errors.Errorf(
		"refreshing access token returned an error (%s) with description: %s",
		tokenErrorBody.Error,
		tokenErrorBody.ErrorDescription,
	)
}

func getAuthFilePath() string {
	return xdg.StateSubpath(filepath.FromSlash("devbox/auth.json"))
}
