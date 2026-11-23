// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nix

import "testing"

func TestHostSystem(t *testing.T) {
	cases := []struct{ goos, goarch, want string }{
		{"linux", "amd64", "x86_64-linux"},
		{"linux", "arm64", "aarch64-linux"},
		{"darwin", "amd64", "x86_64-darwin"},
		{"darwin", "arm64", "aarch64-darwin"},
		{"linux", "riscv64", "riscv64-linux"},
	}
	for _, c := range cases {
		if got := hostSystem(c.goos, c.goarch); got != c.want {
			t.Errorf("hostSystem(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestSystemOrHostNotEmpty(t *testing.T) {
	if SystemOrHost() == "" {
		t.Error("SystemOrHost() returned an empty system")
	}
}
