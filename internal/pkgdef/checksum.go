// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// Digest algorithms.
const (
	algoSHA256 = "sha256"
	algoSHA512 = "sha512"
)

// digest is a parsed checksum: an algorithm and the lowercase hex digest.
// The zero value means "no checksum".
type digest struct {
	algo string
	hex  string
}

// apply records d on the artifact.
func (d digest) apply(a *Artifact) {
	switch d.algo {
	case algoSHA256:
		a.Sha256 = d.hex
	case algoSHA512:
		a.Sha512 = d.hex
	}
}

// fetchChecksum returns the digest for the artifact described by vars, or
// the zero digest when the checksum type is "none" or the vendor publishes
// no digest for this artifact. It returns ErrNoArtifact when the checksum
// source shows that the artifact does not exist.
func (c *Client) fetchChecksum(ctx context.Context, def *Definition, src *Checksum, vars Vars) (digest, error) {
	if src == nil || src.Type == ChecksumNone {
		return digest{}, nil
	}
	if src.Type == ChecksumGitHubRelease {
		return c.fetchGitHubReleaseDigest(ctx, src, vars)
	}
	rawURL, err := Render(src.URL, vars)
	if err != nil {
		return digest{}, err
	}
	rawURL = def.ResolveURL(rawURL)
	switch src.Type {
	case ChecksumHTTPJSON:
		return c.fetchChecksumHTTPJSON(ctx, src, rawURL, vars)
	case ChecksumSHA256URL, ChecksumSHA512URL:
		body, err := c.getCached(ctx, rawURL)
		if errors.Is(err, ErrNotFound) {
			return digest{}, fmt.Errorf("checksum: %w: %w", ErrNoArtifact, err)
		}
		if err != nil {
			return digest{}, fmt.Errorf("checksum: %w", err)
		}
		algo := algoSHA256
		if src.Type == ChecksumSHA512URL {
			algo = algoSHA512
		}
		return parseChecksumFile(string(body), vars.Filename, rawURL, algo)
	default:
		return digest{}, fmt.Errorf("checksum.type %q is not supported", src.Type)
	}
}

func (c *Client) fetchChecksumHTTPJSON(ctx context.Context, src *Checksum, rawURL string, vars Vars) (digest, error) {
	expr, err := Render(src.Select, vars)
	if err != nil {
		return digest{}, err
	}
	results, err := c.queryJSON(ctx, rawURL, expr)
	if err != nil {
		return digest{}, fmt.Errorf("checksum: %w", err)
	}
	for _, r := range results {
		if s, ok := r.(string); ok && s != "" {
			return parseDigest(s, rawURL, "")
		}
	}
	return digest{}, fmt.Errorf("checksum: %w: %s: jq expression %q matched nothing for %s",
		ErrNoArtifact, rawURL, expr, vars.Filename)
}

// githubRelease is the subset of the GitHub release API response used for
// asset digests.
type githubReleaseAssets struct {
	Assets []struct {
		Name   string  `json:"name"`
		Digest *string `json:"digest"`
	} `json:"assets"`
}

// fetchGitHubReleaseDigest reads the digest GitHub computed for a release
// asset. Releases published before GitHub started computing digests have
// none, which is reported as the zero digest (trust on first use).
func (c *Client) fetchGitHubReleaseDigest(ctx context.Context, src *Checksum, vars Vars) (digest, error) {
	tagTemplate := src.Tag
	if tagTemplate == "" {
		tagTemplate = "v{{.Version}}"
	}
	tag, err := Render(tagTemplate, vars)
	if err != nil {
		return digest{}, err
	}
	body, err := c.getGitHubAPI(ctx, "repos/"+src.Repo+"/releases/tags/"+url.PathEscape(tag))
	if errors.Is(err, ErrNotFound) {
		return digest{}, fmt.Errorf("checksum: %w: %w", ErrNoArtifact, err)
	}
	if err != nil {
		return digest{}, fmt.Errorf("checksum: github release %s %s: %w", src.Repo, tag, err)
	}
	var release githubReleaseAssets
	if err := json.Unmarshal(body, &release); err != nil {
		return digest{}, fmt.Errorf("checksum: github release %s %s: response is not JSON: %w", src.Repo, tag, err)
	}
	for _, asset := range release.Assets {
		if asset.Name != vars.Filename {
			continue
		}
		if asset.Digest == nil || *asset.Digest == "" {
			return digest{}, nil
		}
		return parseDigest(*asset.Digest, src.Repo+" "+tag, "")
	}
	return digest{}, fmt.Errorf("checksum: %w: github release %s %s has no asset %s",
		ErrNoArtifact, src.Repo, tag, vars.Filename)
}

// parseChecksumFile extracts the digest for filename from a checksum file.
// The body is either a bare digest or a sha256sum-style list of
// "<hex>  <filename>" lines (a leading "*" marks binary mode). A list
// without an entry for filename means the artifact does not exist.
func parseChecksumFile(body, filename, source, algo string) (digest, error) {
	fields := strings.Fields(body)
	if len(fields) == 1 {
		return parseDigest(fields[0], source, algo)
	}
	for line := range strings.SplitSeq(body, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == filename || strings.HasSuffix(name, "/"+filename) {
			return parseDigest(parts[0], source, algo)
		}
	}
	return digest{}, fmt.Errorf("checksum: %w: %s has no entry for %s", ErrNoArtifact, source, filename)
}

// parseDigest parses a digest in one of the forms vendors publish: bare hex
// (the algorithm follows from the length), "sha256:<hex>", or an SRI string
// such as "sha512-<base64>". When want is not empty the digest must use
// that algorithm.
func parseDigest(value, source, want string) (digest, error) {
	value = strings.TrimSpace(value)
	invalid := fmt.Errorf("checksum: %s: %q is not a sha256 or sha512 digest", source, value)

	var parsed digest
	switch {
	case strings.HasPrefix(value, "sha256:"), strings.HasPrefix(value, "sha512:"):
		algo, hexDigest, _ := strings.Cut(value, ":")
		parsed = digest{algo: algo, hex: strings.ToLower(hexDigest)}
	case strings.HasPrefix(value, "sha256-"), strings.HasPrefix(value, "sha512-"):
		// SRI allows several space-separated hashes; the first is enough.
		first, _, _ := strings.Cut(value, " ")
		algo, b64, _ := strings.Cut(first, "-")
		raw, err := base64.StdEncoding.DecodeString(b64)
		if err != nil {
			return digest{}, invalid
		}
		parsed = digest{algo: algo, hex: hex.EncodeToString(raw)}
	default:
		parsed = digest{hex: strings.ToLower(value)}
		switch len(value) {
		case 64:
			parsed.algo = algoSHA256
		case 128:
			parsed.algo = algoSHA512
		}
	}

	wantLen := map[string]int{algoSHA256: 64, algoSHA512: 128}[parsed.algo]
	if wantLen == 0 || len(parsed.hex) != wantLen {
		return digest{}, invalid
	}
	if _, err := hex.DecodeString(parsed.hex); err != nil {
		return digest{}, invalid
	}
	if want != "" && parsed.algo != want {
		return digest{}, fmt.Errorf("checksum: %s: expected a %s digest, got %s", source, want, parsed.algo)
	}
	return parsed, nil
}
