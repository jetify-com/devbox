// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"testing"

	"go.jetify.com/devbox/internal/envir"
)

func TestFetchClosureStore(t *testing.T) {
	const customCache = "https://nix-cache.example.com/mirror"

	tests := []struct {
		name     string
		cacheURI string
		envValue string
		want     string
	}{
		{
			name:     "https cache uri is used directly",
			cacheURI: "https://cache.nixos.org",
			want:     "https://cache.nixos.org",
		},
		{
			name:     "http cache uri is used directly",
			cacheURI: "http://nix-cache.example.com",
			want:     "http://nix-cache.example.com",
		},
		{
			name:     "https mirror cache uri is used directly",
			cacheURI: customCache,
			want:     customCache,
		},
		{
			name:     "s3 cache falls back to default binary cache",
			cacheURI: "s3://my-bucket",
			want:     "https://cache.nixos.org",
		},
		{
			name:     "s3 cache falls back to configured binary cache",
			cacheURI: "s3://my-bucket",
			envValue: customCache,
			want:     customCache,
		},
		{
			name:     "empty cache falls back to default binary cache",
			cacheURI: "",
			want:     "https://cache.nixos.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv(envir.DevboxNixBinaryCache, tt.envValue)
			}
			if got := fetchClosureStore(tt.cacheURI); got != tt.want {
				t.Errorf("fetchClosureStore(%q) = %q, want %q", tt.cacheURI, got, tt.want)
			}
		})
	}
}
