// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/itchyny/gojq"
	"go.jetify.com/devbox/internal/build"
	"go.jetify.com/devbox/internal/xdg"
)

// ErrNotFound is returned when an HTTP resource responds with 404.
var ErrNotFound = errors.New("not found")

// ErrNotImplemented is returned for install strategies and version sources
// that the spec defines but this version of Devbox does not implement yet.
var ErrNotImplemented = errors.New("not implemented")

// Client resolves, locks and installs package definitions. The zero value
// is not usable; use NewClient.
type Client struct {
	// HTTPClient is used for every network request.
	HTTPClient *http.Client

	// CacheDir is the root under which packages are installed
	// (<CacheDir>/<name>/<version>/<system>) and where remote definitions
	// and downloads are cached.
	CacheDir string

	// GitHubAPIURL is the base URL of the GitHub REST API. Tests point it
	// at an httptest server.
	GitHubAPIURL string

	// GitHubToken, when set, authenticates GitHub API requests to raise
	// the rate limit.
	GitHubToken string

	// Stderr receives the output of post_install hooks. Defaults to
	// io.Discard.
	Stderr io.Writer
}

// NewClient returns a client that caches under $XDG_CACHE_HOME/devbox/pkgs
// and reads GITHUB_TOKEN (or the legacy DEVBOX_GITHUB_API_TOKEN, as runx
// packages do) for GitHub API requests.
func NewClient() *Client {
	return &Client{
		HTTPClient:   &http.Client{Timeout: 10 * time.Minute},
		CacheDir:     xdg.CacheSubpath(filepath.Join("devbox", "pkgs")),
		GitHubAPIURL: "https://api.github.com",
		GitHubToken:  githubTokenFromEnv(),
		Stderr:       io.Discard,
	}
}

func githubTokenFromEnv() string {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token
	}
	return os.Getenv("DEVBOX_GITHUB_API_TOKEN")
}

// InstallDir returns the directory a version of a package is installed to
// on a system. It does not check whether the package is installed.
func (c *Client) InstallDir(name, version, system string) string {
	return filepath.Join(c.CacheDir, name, version, system)
}

func (c *Client) userAgent() string {
	return fmt.Sprintf("Devbox/%s (%s; %s)", build.Version, runtime.GOOS, runtime.GOARCH)
}

// get fetches a URL into memory. file:// URLs are read from disk so that
// tests and local mirrors work without a server. No credentials are ever
// attached: definitions are untrusted input and may point anywhere.
func (c *Client) get(ctx context.Context, rawURL string) ([]byte, error) {
	return c.getWithAuth(ctx, rawURL, "")
}

// getGitHubAPI fetches a GitHub REST API path (relative to GitHubAPIURL)
// with the configured token, if any.
func (c *Client) getGitHubAPI(ctx context.Context, apiPath string) ([]byte, error) {
	endpoint := strings.TrimSuffix(c.GitHubAPIURL, "/") + "/" + strings.TrimPrefix(apiPath, "/")
	auth := ""
	if c.GitHubToken != "" {
		auth = "Bearer " + c.GitHubToken
	}
	return c.getWithAuth(ctx, endpoint, auth)
}

func (c *Client) getWithAuth(ctx context.Context, rawURL, authorization string) ([]byte, error) {
	if p, ok := strings.CutPrefix(rawURL, "file://"); ok {
		b, err := os.ReadFile(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", rawURL, ErrNotFound)
		}
		return b, err
	}

	body, err := c.open(ctx, rawURL, authorization)
	if err != nil {
		return nil, err
	}
	defer body.Close()
	return io.ReadAll(body)
}

// open starts a GET request and returns the response body. The caller must
// close it. Non-2xx responses are returned as errors that include the start
// of the body, which is usually a short explanation.
func (c *Client) open(ctx context.Context, rawURL, authorization string) (io.ReadCloser, error) {
	if p, ok := strings.CutPrefix(rawURL, "file://"); ok {
		f, err := os.Open(p)
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("%s: %w", rawURL, ErrNotFound)
		}
		return f, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", c.userAgent())
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	res, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode == http.StatusNotFound {
		res.Body.Close()
		return nil, fmt.Errorf("GET %s: %w", rawURL, ErrNotFound)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		snippet, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		res.Body.Close()
		msg := strings.TrimSpace(string(bytes.ToValidUTF8(snippet, nil)))
		if msg == "" {
			return nil, fmt.Errorf("GET %s: %s", rawURL, res.Status)
		}
		return nil, fmt.Errorf("GET %s: %s: %s", rawURL, res.Status, msg)
	}
	return res.Body, nil
}

func compileJQ(expr string) (*gojq.Code, error) {
	query, err := gojq.Parse(expr)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression %q: %w", expr, err)
	}
	code, err := gojq.Compile(query)
	if err != nil {
		return nil, fmt.Errorf("invalid jq expression %q: %w", expr, err)
	}
	return code, nil
}

// runJQ evaluates a compiled jq expression against input and returns every
// string it emits. Numbers are formatted as strings; arrays are flattened
// one level so that ".[].version" and "[.[].version]" both work.
func runJQ(code *gojq.Code, expr string, input any) ([]string, error) {
	var out []string
	iter := code.Run(input)
	for {
		value, ok := iter.Next()
		if !ok {
			break
		}
		if err, isErr := value.(error); isErr {
			return nil, fmt.Errorf("jq expression %q: %w", expr, err)
		}
		if arr, isArr := value.([]any); isArr {
			for _, elem := range arr {
				out = appendJQValue(out, elem)
			}
			continue
		}
		out = appendJQValue(out, value)
	}
	return out, nil
}

func appendJQValue(out []string, v any) []string {
	switch val := v.(type) {
	case string:
		return append(out, val)
	case float64:
		// encoding/json decodes every number as float64. Format without an
		// exponent so that 20240101 stays "20240101" and not "2.0240101e+07".
		return append(out, strconv.FormatFloat(val, 'f', -1, 64))
	case int:
		return append(out, strconv.Itoa(val))
	case nil, bool:
		return out
	default:
		return append(out, fmt.Sprint(val))
	}
}
