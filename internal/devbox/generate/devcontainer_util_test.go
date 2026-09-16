// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package generate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestCreateDockerfileDevOmitsNixStoreOptimise ensures the generated dev
// Dockerfile does not run `nix-store --optimise`. That command fails on Docker's
// default overlay2 storage driver ("cannot rename: Stale file handle") and
// breaks the image build, while providing negligible size savings in a freshly
// built image. Regression test for
// https://github.com/jetify-com/devbox/issues/2616.
func TestCreateDockerfileDevOmitsNixStoreOptimise(t *testing.T) {
	for _, rootUser := range []bool{false, true} {
		dockerfile := generateDevDockerfile(t, rootUser)

		if strings.Contains(dockerfile, "nix-store --optimise") {
			t.Errorf("generated dev Dockerfile (rootUser=%t) should not run "+
				"`nix-store --optimise`; it breaks Docker builds on overlay2.\n%s",
				rootUser, dockerfile)
		}
		// The garbage-collection pass is still wanted: it removes build-time
		// dependencies and is what actually shrinks the image.
		if !strings.Contains(dockerfile, "nix-store --gc") {
			t.Errorf("generated dev Dockerfile (rootUser=%t) should still run "+
				"`nix-store --gc`.\n%s", rootUser, dockerfile)
		}
	}
}

func generateDevDockerfile(t *testing.T, rootUser bool) string {
	t.Helper()

	dir := t.TempDir()
	g := &Options{
		Path:     dir,
		RootUser: rootUser,
	}
	if err := g.CreateDockerfile(t.Context(), CreateDockerfileOptions{
		ForType: "dev",
	}); err != nil {
		t.Fatalf("CreateDockerfile failed: %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
	if err != nil {
		t.Fatalf("reading generated Dockerfile: %v", err)
	}
	return string(contents)
}

// TestCreateDockerfileDevCopiesConfigFileName ensures the generated dev
// Dockerfile copies the project's actual config file, which may be named
// devbox.jsonc rather than devbox.json.
func TestCreateDockerfileDevCopiesConfigFileName(t *testing.T) {
	for _, name := range []string{"", "devbox.json", "devbox.jsonc"} {
		dir := t.TempDir()
		g := &Options{Path: dir, ConfigFileName: name}
		if err := g.CreateDockerfile(t.Context(), CreateDockerfileOptions{ForType: "dev"}); err != nil {
			t.Fatalf("CreateDockerfile(ConfigFileName=%q) failed: %v", name, err)
		}
		contents, err := os.ReadFile(filepath.Join(dir, "Dockerfile"))
		if err != nil {
			t.Fatal(err)
		}
		want := name
		if want == "" {
			want = "devbox.json"
		}
		// Non-root images use `COPY --chown=... <src> <dst>`; match on the
		// trailing "<src> <dst>" pair so both forms are covered.
		if !strings.Contains(string(contents), " "+want+" "+want+"\n") {
			t.Errorf("ConfigFileName=%q: Dockerfile should copy %s, got:\n%s", name, want, contents)
		}
		if want != "devbox.json" && strings.Contains(string(contents), " devbox.json devbox.json") {
			t.Errorf("ConfigFileName=%q: Dockerfile should not also copy devbox.json, got:\n%s", name, contents)
		}
	}
}
