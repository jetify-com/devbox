// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"go.jetify.com/pkg/filecache"
)

func TestFetchNarInfoStatusFromHTTP(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())

	t.Run("hit", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodHead || r.URL.Path != "/abc.narinfo" {
				t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		inCache, err := fetchNarInfoStatusFromHTTP(t.Context(), srv.URL, "abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !inCache {
			t.Error("expected package to be in cache")
		}
	})

	t.Run("miss", func(t *testing.T) {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer srv.Close()

		inCache, err := fetchNarInfoStatusFromHTTP(t.Context(), srv.URL, "abc")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if inCache {
			t.Error("expected package to not be in cache")
		}
	})

	t.Run("timeout is treated as a miss", func(t *testing.T) {
		release := make(chan struct{})
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-release
		}))
		defer srv.Close()
		defer close(release)

		orig := narInfoHTTPTimeout
		narInfoHTTPTimeout = 50 * time.Millisecond
		defer func() { narInfoHTTPTimeout = orig }()

		inCache, err := fetchNarInfoStatusFromHTTP(t.Context(), srv.URL, "abc")
		if err != nil {
			t.Fatalf("timeout should not be an error, got: %v", err)
		}
		if inCache {
			t.Error("expected package to not be in cache")
		}
	})

	t.Run("unreachable cache is treated as a miss", func(t *testing.T) {
		srv := httptest.NewServer(http.NotFoundHandler())
		url := srv.URL
		srv.Close() // connection refused from here on

		inCache, err := fetchNarInfoStatusFromHTTP(t.Context(), url, "abc")
		if err != nil {
			t.Fatalf("connection error should not be an error, got: %v", err)
		}
		if inCache {
			t.Error("expected package to not be in cache")
		}
	})
}

// narInfoServer starts a binary cache that responds with status to every
// narinfo request and counts the requests it receives.
func narInfoServer(t *testing.T, status int) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.WriteHeader(status)
	}))
	t.Cleanup(srv.Close)
	return srv, &requests
}

// fetchNarInfo calls fetchNarInfoStatusFromHTTP and fails the test on error.
func fetchNarInfo(t *testing.T, uri, hash string) bool {
	t.Helper()
	inCache, err := fetchNarInfoStatusFromHTTP(t.Context(), uri, hash)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	return inCache
}

// clearNarInfoMemoryCache simulates a new devbox process by dropping the
// in-memory layer of the narinfo cache.
func clearNarInfoMemoryCache(t *testing.T) {
	t.Helper()
	narInfoStatusFnCache.Clear()
	t.Cleanup(narInfoStatusFnCache.Clear)
}

func TestFetchNarInfoStatusCache(t *testing.T) {
	t.Run("memory caches hits and misses", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		clearNarInfoMemoryCache(t)
		hitSrv, hitRequests := narInfoServer(t, http.StatusOK)
		missSrv, missRequests := narInfoServer(t, http.StatusNotFound)

		for range 3 {
			if !fetchNarInfo(t, hitSrv.URL, "abc") {
				t.Error("expected hit")
			}
			if fetchNarInfo(t, missSrv.URL, "abc") {
				t.Error("expected miss")
			}
		}
		if got := hitRequests.Load(); got != 1 {
			t.Errorf("got %d requests for hit, want 1", got)
		}
		if got := missRequests.Load(); got != 1 {
			t.Errorf("got %d requests for miss, want 1", got)
		}
	})

	t.Run("hits persist on disk", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		clearNarInfoMemoryCache(t)
		srv, requests := narInfoServer(t, http.StatusOK)

		if !fetchNarInfo(t, srv.URL, "abc") {
			t.Fatal("expected hit")
		}
		clearNarInfoMemoryCache(t)
		if !fetchNarInfo(t, srv.URL, "abc") {
			t.Error("expected hit from disk cache")
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("got %d requests, want 1", got)
		}

		// A different hash in the same cache is a different entry.
		if !fetchNarInfo(t, srv.URL, "def") {
			t.Error("expected hit")
		}
		if got := requests.Load(); got != 2 {
			t.Errorf("got %d requests, want 2", got)
		}
	})

	t.Run("misses are not persisted", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		clearNarInfoMemoryCache(t)
		srv, requests := narInfoServer(t, http.StatusNotFound)

		if fetchNarInfo(t, srv.URL, "abc") {
			t.Fatal("expected miss")
		}
		clearNarInfoMemoryCache(t)
		if fetchNarInfo(t, srv.URL, "abc") {
			t.Error("expected miss")
		}
		if got := requests.Load(); got != 2 {
			t.Errorf("got %d requests, want 2", got)
		}
		if _, err := narInfoDiskCache().Get(srv.URL + "/abc"); !filecache.IsCacheMiss(err) {
			t.Errorf("expected no disk cache entry for a miss, got err = %v", err)
		}
	})

	t.Run("expired hits are refetched", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		clearNarInfoMemoryCache(t)
		srv, requests := narInfoServer(t, http.StatusOK)

		key := srv.URL + "/abc"
		if err := narInfoDiskCache().Set(key, true, -time.Second); err != nil {
			t.Fatal(err)
		}
		if !fetchNarInfo(t, srv.URL, "abc") {
			t.Error("expected hit")
		}
		if got := requests.Load(); got != 1 {
			t.Errorf("got %d requests, want 1", got)
		}
		if _, err := narInfoDiskCache().Get(key); err != nil {
			t.Errorf("expected refreshed disk cache entry, got err = %v", err)
		}
	})

	t.Run("hits don't leak between caches", func(t *testing.T) {
		t.Setenv("XDG_CACHE_HOME", t.TempDir())
		clearNarInfoMemoryCache(t)
		hitSrv, _ := narInfoServer(t, http.StatusOK)
		missSrv, missRequests := narInfoServer(t, http.StatusNotFound)

		if !fetchNarInfo(t, hitSrv.URL, "abc") {
			t.Fatal("expected hit")
		}
		clearNarInfoMemoryCache(t)
		if fetchNarInfo(t, missSrv.URL, "abc") {
			t.Error("hit in one cache was reported for another")
		}
		if got := missRequests.Load(); got != 1 {
			t.Errorf("got %d requests to second cache, want 1", got)
		}
	})

	t.Run("disk cache write failure is not an error", func(t *testing.T) {
		// Point the cache dir at a regular file so that writes fail.
		notADir := filepath.Join(t.TempDir(), "file")
		if err := os.WriteFile(notADir, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_CACHE_HOME", notADir)
		clearNarInfoMemoryCache(t)
		srv, _ := narInfoServer(t, http.StatusOK)

		if !fetchNarInfo(t, srv.URL, "abc") {
			t.Error("expected hit")
		}
		if _, err := narInfoDiskCache().Get(srv.URL + "/abc"); err == nil {
			t.Error("expected disk cache to be unusable in this test")
		}
	})
}
