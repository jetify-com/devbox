// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/nix"
)

func TestFetchResolvedPackageSearchErrors(t *testing.T) {
	tests := []struct {
		status       int
		wantNotFound bool
	}{
		{http.StatusNotFound, true},
		{http.StatusInternalServerError, false},
	}
	for _, test := range tests {
		t.Run(http.StatusText(test.status), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(test.status)
			}))
			defer srv.Close()
			t.Setenv(envir.DevboxSearchHost, srv.URL)
			t.Setenv("DEVBOX_FEATURE_RESOLVE_V2", "0")

			_, err := (&File{}).FetchResolvedPackage("hello@1.0", false)
			if err == nil {
				t.Fatal("got nil error")
			}
			if got := errors.Is(err, nix.ErrPackageNotFound); got != test.wantNotFound {
				t.Errorf("errors.Is(err, ErrPackageNotFound) = %v, want %v (err: %v)", got, test.wantNotFound, err)
			}
		})
	}
}

func TestFetchResolvedPackagePinnedGitHubFlake(t *testing.T) {
	const pkg = "github:owner/repo/1111111111111111111111111111111111111111#hello"
	for _, refresh := range []bool{false, true} {
		got, err := (&File{}).FetchResolvedPackage(pkg, refresh)
		if err != nil {
			t.Fatalf("refresh=%v: got error: %v", refresh, err)
		}
		if got.Resolved != pkg {
			t.Errorf("refresh=%v: got Resolved %q, want %q", refresh, got.Resolved, pkg)
		}
		if got.LastModified != "" {
			t.Errorf("refresh=%v: got LastModified %q, want empty", refresh, got.LastModified)
		}
	}
}
