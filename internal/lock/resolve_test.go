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

func TestFetchResolvedPackageErrors(t *testing.T) {
	for _, resolveV2 := range []string{"0", "1"} {
		for _, test := range []struct {
			status       int
			wantNotFound bool
		}{
			{http.StatusNotFound, true},
			{http.StatusInternalServerError, false},
		} {
			t.Run("v2="+resolveV2+"/"+http.StatusText(test.status), func(t *testing.T) {
				srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(test.status)
				}))
				t.Cleanup(srv.Close)
				t.Setenv(envir.DevboxSearchHost, srv.URL)
				t.Setenv("DEVBOX_FEATURE_RESOLVE_V2", resolveV2)

				_, err := (&File{}).FetchResolvedPackage("hello@latest", false)
				if err == nil {
					t.Fatal("got nil error, want an error")
				}
				if got := errors.Is(err, nix.ErrPackageNotFound); got != test.wantNotFound {
					t.Errorf("errors.Is(err, nix.ErrPackageNotFound) = %v, want %v (err: %v)", got, test.wantNotFound, err)
				}
			})
		}
	}
}
