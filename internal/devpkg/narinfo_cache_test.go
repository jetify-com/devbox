// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestFetchNarInfoStatusFromHTTP(t *testing.T) {
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
