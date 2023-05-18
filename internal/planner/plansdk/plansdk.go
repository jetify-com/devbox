// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"go.jetpack.io/devbox/internal/envir"
)

// FlakePlan contains the data to populate the top level flake.nix file
// that builds the devbox environment
type FlakePlan struct {
	NixpkgsInfo *NixpkgsInfo
	FlakeInputs []*FlakeInput
}

// Contains default nixpkgs used for mkShell
type NixpkgsInfo struct {
	URL    string
	TarURL string
}

// The commit hash for nixpkgs-unstable on 2023-01-25 from status.nixos.org
const DefaultNixpkgsCommit = "f80ac848e3d6f0c12c52758c0f25c10c97ca3b62"

func GetNixpkgsInfo(commitHash string) *NixpkgsInfo {
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
