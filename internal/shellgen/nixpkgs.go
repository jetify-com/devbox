// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.jetify.com/devbox/internal/envir"
)

// Contains default nixpkgs used for mkShell
type NixpkgsInfo struct {
	URL    string
	TarURL string
}

func getNixpkgsInfo(commitHash string) *NixpkgsInfo {
	url := fmt.Sprintf("github:NixOS/nixpkgs/%s", commitHash)
	if mirror := nixpkgsMirrorURL(commitHash); mirror != "" {
		url = mirror
	}
	return &NixpkgsInfo{
		URL: url,
		// legacy, used for shell.nix (which is no longer used, but some direnv users still need it)
		TarURL: fmt.Sprintf("https://github.com/nixos/nixpkgs/archive/%s.tar.gz", commitHash),
	}
}

func nixpkgsMirrorURL(commitHash string) string {
	baseURL := os.Getenv(envir.DevboxCache)
	if baseURL == "" {
		return ""
	}

	// Check that the mirror is responsive and has the tar file. We can't
	// leave this up to Nix because fetchTarball will retry indefinitely.
	client := &http.Client{Timeout: 3 * time.Second}
	mirrorURL := fmt.Sprintf("%s/nixos/nixpkgs/archive/%s.tar.gz", baseURL, commitHash)
	resp, err := client.Head(mirrorURL)
	if err != nil || resp.StatusCode != http.StatusOK {
		return ""
	}
	return mirrorURL
}
