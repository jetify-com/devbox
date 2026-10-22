// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const minimalDef = `{
  "schema": 1,
  "name": "hello",
  "platforms": ["aarch64-darwin", "x86_64-linux"],
  "versions": {"type": "static", "versions": ["1.0.0", "1.1.0"]},
  "install": {"type": "archive", "url": "https://example.com/hello-{{.Version}}-{{.OS}}-{{.Arch}}.tar.gz"}
}`

func TestParseGoExample(t *testing.T) {
	content, err := os.ReadFile(filepath.Join("..", "..", "pkgs", "go", "go.pkg.json"))
	require.NoError(t, err)

	def, err := Parse(content)
	require.NoError(t, err)
	assert.Equal(t, "go", def.Name)
	assert.Equal(t, VersionsHTTPJSON, def.Versions.Type)
	assert.Equal(t, InstallArchive, def.Install.Type)
	assert.Equal(t, ChecksumHTTPJSON, def.Install.ChecksumType())
	assert.Equal(t, []string{"bin"}, def.BinDirs())
	assert.Equal(t, []string{"go", "gofmt"}, def.Binaries)
	assert.False(t, def.UsesScripts())
	require.NotNil(t, def.Install.StripComponents)
	assert.Equal(t, 1, *def.Install.StripComponents)
}

func TestParseMinimalWithComments(t *testing.T) {
	def, err := Parse([]byte("// comment\n" + minimalDef))
	require.NoError(t, err)
	assert.Equal(t, "hello", def.Name)
	assert.Equal(t, ChecksumNone, def.Install.ChecksumType())
	assert.True(t, def.SupportsSystem("x86_64-linux"))
	assert.False(t, def.SupportsSystem("aarch64-linux"))
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name string
		json string
		want string
	}{
		{
			name: "not json",
			json: `{`,
			want: "not valid JSON",
		},
		{
			name: "wrong schema",
			json: `{"schema": 2, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u"}}`,
			want: "schema must be 1",
		},
		{
			name: "bad name",
			json: `{"schema": 1, "name": "../x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u"}}`,
			want: "name \"../x\" must match",
		},
		{
			name: "no platforms",
			json: `{"schema": 1, "name": "x", "platforms": [], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u"}}`,
			want: "platforms must list at least one system",
		},
		{
			name: "unknown platform",
			json: `{"schema": 1, "name": "x", "platforms": ["amd64-windows"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u"}}`,
			want: "unknown system \"amd64-windows\"",
		},
		{
			name: "missing versions type",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {}, "install": {"type": "archive", "url": "u"}}`,
			want: "versions.type is required",
		},
		{
			name: "bad jq",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "http_json", "url": "u", "select": ".[ |"}, "install": {"type": "archive", "url": "u"}}`,
			want: "versions.select: invalid jq expression",
		},
		{
			name: "github repo format",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "github_releases", "repo": "nope"}, "install": {"type": "archive", "url": "u"}}`,
			want: "versions.repo must be of the form owner/name",
		},
		{
			name: "github repo traversal",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "github_releases", "repo": "owner/../../user"}, "install": {"type": "archive", "url": "u"}}`,
			want: "versions.repo must be of the form owner/name",
		},
		{
			name: "github repo extra segment",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "github_releases", "repo": "owner/name/extra"}, "install": {"type": "archive", "url": "u"}}`,
			want: "versions.repo must be of the form owner/name",
		},
		{
			name: "install url missing",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "binary"}}`,
			want: "install.url is required for type binary",
		},
		{
			name: "unknown install type",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "magic"}}`,
			want: "install.type \"magic\" is not one of",
		},
		{
			name: "checksum type",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u", "checksum": {"type": "md5"}}}`,
			want: "install.checksum.type \"md5\" is not one of",
		},
		{
			name: "override for unlisted platform",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u", "overrides": {"aarch64-darwin": {"url": "v"}}}}`,
			want: "install.overrides: \"aarch64-darwin\" is not in platforms",
		},
		{
			name: "nested override",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u", "overrides": {"x86_64-linux": {"overrides": {"x86_64-linux": {}}}}}}`,
			want: "install.overrides.x86_64-linux may not contain overrides",
		},
		{
			name: "bin escapes root",
			json: `{"schema": 1, "name": "x", "platforms": ["x86_64-linux"], "versions": {"type": "static", "versions": ["1"]}, "install": {"type": "archive", "url": "u"}, "bin": ["../bin"]}`,
			want: "bin: \"../bin\" must be a relative path",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(tt.json))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestValidateReportsEveryError(t *testing.T) {
	_, err := Parse([]byte(`{"schema": 1, "platforms": [], "versions": {}, "install": {}}`))
	require.Error(t, err)
	msg := err.Error()
	for _, want := range []string{"name is required", "platforms must list", "versions.type is required", "install.type is required"} {
		assert.Contains(t, msg, want)
	}
	assert.Equal(t, 4, strings.Count(msg, "\n  - "))
}

func TestInstallFor(t *testing.T) {
	def, err := Parse([]byte(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux", "aarch64-darwin"],
	  "versions": {"type": "static", "versions": ["1"]},
	  "install": {
	    "type": "archive", "url": "default", "strip_components": 1,
	    "checksum": {"type": "sha256_url", "url": "{{.URL}}.sha256"},
	    "overrides": {"aarch64-darwin": {"type": "binary", "url": "mac", "checksum": {"type": "none"}}}
	  }
	}`))
	require.NoError(t, err)

	linux := def.InstallFor("x86_64-linux")
	assert.Equal(t, InstallArchive, linux.Type)
	assert.Equal(t, "default", linux.URL)
	assert.Equal(t, ChecksumSHA256URL, linux.ChecksumType())
	assert.Nil(t, linux.Overrides)

	mac := def.InstallFor("aarch64-darwin")
	assert.Equal(t, InstallBinary, mac.Type)
	assert.Equal(t, "mac", mac.URL)
	assert.Equal(t, ChecksumNone, mac.ChecksumType())
	require.NotNil(t, mac.StripComponents, "unset override fields keep the default")
	assert.Equal(t, 1, *mac.StripComponents)

	// The definition itself is not mutated.
	assert.Len(t, def.Install.Overrides, 1)
}

func TestUsesScripts(t *testing.T) {
	def, err := Parse([]byte(minimalDef))
	require.NoError(t, err)
	assert.False(t, def.UsesScripts())

	def.Hooks.PostInstall = []string{"true"}
	assert.True(t, def.UsesScripts())
}
