// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package lock

import "testing"

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
