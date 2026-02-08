// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"fmt"
	"net/http"
	"os"
	"time"
	"log/slog"

	"go.jetify.com/devbox/internal/envir"
)

// Contains default nixpkgs used for mkShell
type NixpkgsInfo struct {
	URL    string
	TarURL string
}

func getNixpkgsInfo(commitHash string) *NixpkgsInfo {
	// Default URLs pointing to GitHub
	url := fmt.Sprintf("github:NixOS/nixpkgs/%s", commitHash)
	tarURL := fmt.Sprintf("https://github.com/nixos/nixpkgs/archive/%s.tar.gz", commitHash)

	// Check if a local/internal mirror is available via DEVBOX_CACHE
	if mirror := nixpkgsMirrorURL(commitHash); mirror != "" {
		url = mirror      // flakes use this URL
		tarURL = mirror   // devbox shell uses this tarball URL
		slog.Debug("Using local/internal mirror for nixpkgs", "commit", commitHash, "url", url, "tarURL", tarURL)
	} else {
		slog.Debug("No local mirror found, falling back to GitHub", "commit", commitHash, "url", url, "tarURL", tarURL)
	}

	return &NixpkgsInfo{
		URL:    url,
		TarURL: tarURL,
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
