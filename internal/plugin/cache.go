// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"go.jetify.com/pkg/filecache"
)

// defaultRemoteCacheTTL is how long content fetched from a remote plugin at a
// branch, tag, or default branch is used before it's fetched again.
const defaultRemoteCacheTTL = 24 * time.Hour

// pinnedCacheTTL is used for content fetched at a full commit hash, which
// can't change. filecache needs an expiration, so use one far in the future.
const pinnedCacheTTL = 100 * 365 * 24 * time.Hour

// remoteCacheTTL returns how long to cache content fetched from a remote
// plugin at the given rev (which may be empty).
func remoteCacheTTL(rev string) (time.Duration, error) {
	if isFullCommitHash(rev) {
		return pinnedCacheTTL, nil
	}

	// This is a stopgap until plugin is stored in lockfile.
	// DEVBOX_X indicates this is an experimental env var.
	// Use DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL to override the default TTL.
	// e.g. DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL=1h will cache the plugin for 1 hour.
	// Note: If you want to disable cache, we recommend using a low second value instead of zero to
	// ensure only one network request is made.
	ttlStr := os.Getenv("DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL")
	if ttlStr == "" {
		return defaultRemoteCacheTTL, nil
	}
	ttl, err := time.ParseDuration(ttlStr)
	if err != nil {
		return 0, fmt.Errorf("invalid DEVBOX_X_GITHUB_PLUGIN_CACHE_TTL=%q: %w", ttlStr, err)
	}
	return ttl, nil
}

// getOrFetch returns the cached value for key if it hasn't expired. Otherwise
// it calls fetch and caches the result for ttl. If fetch fails and an expired
// value is cached, it returns the expired value so that remote plugins keep
// working when the network or the remote is unavailable.
func getOrFetch(
	cache *filecache.Cache[[]byte],
	key string,
	ttl time.Duration,
	fetch func() ([]byte, error),
) ([]byte, error) {
	cached, cacheErr := cache.Get(key)
	if cacheErr == nil {
		return cached, nil
	}
	if !filecache.IsCacheMiss(cacheErr) {
		return nil, cacheErr
	}

	content, err := fetch()
	if err != nil {
		if errors.Is(cacheErr, filecache.Expired) {
			slog.Warn("failed to fetch remote plugin content, using expired cached copy",
				"key", key, "err", err)
			return cached, nil
		}
		return nil, err
	}
	return content, cache.Set(key, content, ttl)
}

// isFullCommitHash reports whether s is a full 40-character commit hash.
// Short hashes, branches, and tags are not, since what they point to can
// change.
func isFullCommitHash(s string) bool {
	return s != "" && !isBranchName(s)
}
