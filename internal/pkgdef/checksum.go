// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

// fetchChecksum returns the lowercase hex sha256 for the artifact described
// by vars, or "" when the checksum type is "none".
func (c *Client) fetchChecksum(ctx context.Context, src *Checksum, vars Vars) (string, error) {
	if src == nil || src.Type == ChecksumNone {
		return "", nil
	}
	rawURL, err := Render(src.URL, vars)
	if err != nil {
		return "", err
	}
	switch src.Type {
	case ChecksumHTTPJSON:
		return c.fetchChecksumHTTPJSON(ctx, src, rawURL, vars)
	case ChecksumSHA256URL:
		body, err := c.get(ctx, rawURL)
		if err != nil {
			return "", fmt.Errorf("checksum: %w", err)
		}
		return parseSHA256Body(string(body), vars.Filename, rawURL)
	default:
		return "", fmt.Errorf("checksum.type %q is not supported", src.Type)
	}
}

func (c *Client) fetchChecksumHTTPJSON(ctx context.Context, src *Checksum, rawURL string, vars Vars) (string, error) {
	expr, err := Render(src.Select, vars)
	if err != nil {
		return "", err
	}
	code, err := compileJQ(expr)
	if err != nil {
		return "", err
	}
	body, err := c.get(ctx, rawURL)
	if err != nil {
		return "", fmt.Errorf("checksum: %w", err)
	}
	var input any
	if err := json.Unmarshal(body, &input); err != nil {
		return "", fmt.Errorf("checksum: %s: response is not JSON: %w", rawURL, err)
	}
	results, err := runJQ(code, expr, input)
	if err != nil {
		return "", fmt.Errorf("checksum: %w", err)
	}
	if len(results) == 0 {
		return "", fmt.Errorf("checksum: %s: jq expression %q matched nothing for %s", rawURL, expr, vars.Filename)
	}
	return normalizeSHA256(results[0], rawURL)
}

// parseSHA256Body extracts the digest for filename from a checksum file.
// The body is either a bare digest or a sha256sum-style list of
// "<hex>  <filename>" lines (a leading "*" marks binary mode).
func parseSHA256Body(body, filename, source string) (string, error) {
	fields := strings.Fields(body)
	if len(fields) == 1 {
		return normalizeSHA256(fields[0], source)
	}
	for line := range strings.SplitSeq(body, "\n") {
		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue
		}
		name := strings.TrimPrefix(parts[1], "*")
		if name == filename || strings.HasSuffix(name, "/"+filename) {
			return normalizeSHA256(parts[0], source)
		}
	}
	return "", fmt.Errorf("checksum: %s has no entry for %s", source, filename)
}

func normalizeSHA256(digest, source string) (string, error) {
	digest = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(digest, "sha256:")))
	if len(digest) != 64 {
		return "", fmt.Errorf("checksum: %s: %q is not a hex sha256", source, digest)
	}
	if _, err := hex.DecodeString(digest); err != nil {
		return "", fmt.Errorf("checksum: %s: %q is not a hex sha256", source, digest)
	}
	return digest, nil
}
