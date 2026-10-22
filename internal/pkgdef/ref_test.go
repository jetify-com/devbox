// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseRef(t *testing.T) {
	tests := []struct {
		raw     string
		want    Ref
		local   bool
		wantErr string
	}{
		{raw: "pkg:./go.pkg.json@1.22", want: Ref{Source: "./go.pkg.json", Version: "1.22"}, local: true},
		{raw: "pkg:./go.pkg.json", want: Ref{Source: "./go.pkg.json"}, local: true},
		{raw: "pkg:../defs/go.pkg.json@latest", want: Ref{Source: "../defs/go.pkg.json", Version: "latest"}, local: true},
		{raw: "pkg:/abs/go.pkg.json@1", want: Ref{Source: "/abs/go.pkg.json", Version: "1"}, local: true},
		{raw: "pkg:file:///tmp/go.pkg.json@1", want: Ref{Source: "file:///tmp/go.pkg.json", Version: "1"}, local: true},
		{raw: "pkg:https://example.com/go.pkg.json@1.22.3", want: Ref{Source: "https://example.com/go.pkg.json", Version: "1.22.3"}},
		{
			// The version splits at the last "@" so sources may contain one.
			raw:  "pkg:https://user@example.com/go.pkg.json@1",
			want: Ref{Source: "https://user@example.com/go.pkg.json", Version: "1"},
		},
		{raw: "go@1.22", wantErr: "missing \"pkg:\" scheme"},
		{raw: "pkg:", wantErr: "missing definition path or URL"},
		{raw: "pkg:@1.22", wantErr: "missing definition path or URL"},
		{raw: "pkg:github:owner/repo@1", wantErr: "unsupported definition source scheme \"github\""},
	}
	for _, test := range tests {
		t.Run(test.raw, func(t *testing.T) {
			got, err := ParseRef(test.raw)
			if test.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), test.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, test.want, got)
			assert.Equal(t, test.raw, got.String())
			assert.Equal(t, test.local, got.IsLocal())
			assert.Equal(t, !test.local, got.IsRemote())
		})
	}
}

func TestIsRef(t *testing.T) {
	assert.True(t, IsRef("pkg:./x.pkg.json"))
	assert.False(t, IsRef("runx:owner/repo"))
	assert.False(t, IsRef("go@1.22"))
	assert.False(t, IsRef("path:./flake"))
}

func TestRefWithVersion(t *testing.T) {
	ref := Ref{Source: "./go.pkg.json", Version: "1.22"}
	assert.Equal(t, "pkg:./go.pkg.json@1.22.3", ref.WithVersion("1.22.3").String())
	assert.Equal(t, "1.22", ref.Version, "WithVersion returns a copy")
}

func TestLocalPath(t *testing.T) {
	assert.Equal(t, filepath.Join("/proj", "go.pkg.json"), Ref{Source: "./go.pkg.json"}.LocalPath("/proj"))
	assert.Equal(t, "/abs/go.pkg.json", Ref{Source: "/abs/go.pkg.json"}.LocalPath("/proj"))
	assert.Equal(t, "/tmp/go.pkg.json", Ref{Source: "file:///tmp/go.pkg.json"}.LocalPath("/proj"))
	assert.Empty(t, Ref{Source: "https://example.com/go.pkg.json"}.LocalPath("/proj"))
}
