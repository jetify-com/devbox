// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devpkg

import (
	"testing"

	"go.jetify.com/devbox/internal/envir"
)

func TestBinaryCache(t *testing.T) {
	t.Run("defaults to the public cache", func(t *testing.T) {
		// t.Setenv guarantees the env var is restored after the test; set it
		// empty to ensure we exercise the default path regardless of the
		// caller's environment.
		t.Setenv(envir.DevboxNixBinaryCache, "")
		if got, want := BinaryCache(), defaultBinaryCache; got != want {
			t.Errorf("BinaryCache() = %q, want default %q", got, want)
		}
	})

	t.Run("is overridable via DEVBOX_NIX_BINARY_CACHE", func(t *testing.T) {
		const mirror = "https://nix-cache.example.com/mirror"
		t.Setenv(envir.DevboxNixBinaryCache, mirror)
		if got := BinaryCache(); got != mirror {
			t.Errorf("BinaryCache() = %q, want override %q", got, mirror)
		}
	})
}
