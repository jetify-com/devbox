// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"encoding/json"
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
	content, err := os.ReadFile(filepath.Join("..", "..", "pkgs", "go", "go.pkg.jsonc"))
	require.NoError(t, err)

	def, err := Parse(content)
	require.NoError(t, err)
	assert.Equal(t, "go", def.Name)
	assert.Equal(t, VersionsHTTPJSON, def.Versions[0].Type)
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
			want: "install.url or .lookup is required for type binary",
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

	linux := def.InstallFor("x86_64-linux", "1.0.0")
	assert.Equal(t, InstallArchive, linux.Type)
	assert.Equal(t, "default", linux.URL)
	assert.Equal(t, ChecksumSHA256URL, linux.ChecksumType())
	assert.Nil(t, linux.Overrides)

	mac := def.InstallFor("aarch64-darwin", "1.0.0")
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

func TestVersionSourcesJSON(t *testing.T) {
	def, err := Parse([]byte(minimalDef))
	require.NoError(t, err)
	require.Len(t, def.Versions, 1)

	// A list of sources is their union.
	def, err = Parse([]byte(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": [
	    {"type": "static", "versions": ["2.0.0"]},
	    {"type": "static", "versions": ["1.0.0"]}
	  ],
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`))
	require.NoError(t, err)
	require.Len(t, def.Versions, 2)

	// A single source round-trips as an object, a list as an array.
	single, err := json.Marshal(VersionSources{{Type: VersionsStatic, Versions: []string{"1"}}})
	require.NoError(t, err)
	assert.JSONEq(t, `{"type": "static", "versions": ["1"]}`, string(single))
	list, err := json.Marshal(def.Versions)
	require.NoError(t, err)
	assert.JSONEq(t, `[{"type": "static", "versions": ["2.0.0"]}, {"type": "static", "versions": ["1.0.0"]}]`, string(list))

	// Each source in a list is validated, with its index in the error.
	_, err = Parse([]byte(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": [{"type": "static", "versions": ["1"]}, {"type": "github_releases"}],
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`))
	require.ErrorContains(t, err, "versions[1].repo must be of the form owner/name")

	_, err = Parse([]byte(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
	  "versions": [],
	  "install": {"type": "binary", "url": "https://example.com/x"}
	}`))
	require.ErrorContains(t, err, "versions is required")
}

func TestInstallForVersionOverrides(t *testing.T) {
	def, err := Parse([]byte(`{
	  "schema": 1, "name": "x", "platforms": ["x86_64-linux", "aarch64-darwin"],
	  "versions": {"type": "static", "versions": ["1"]},
	  "install": {
	    "type": "archive", "url": "new", "strip_components": 1,
	    "checksum": {"type": "github_release", "repo": "acme/x"},
	    "overrides": {"aarch64-darwin": {"url": "new-mac"}},
	    "version_overrides": [
	      {"versions": "<2", "type": "binary", "url": "old", "checksum": {"type": "none"},
	       "overrides": {"aarch64-darwin": {"url": "old-mac"}}},
	      {"versions": "<3", "lookup": {"url": "https://example.com/index.json", "select": ".x"}}
	    ]
	  }
	}`))
	require.NoError(t, err)

	tests := []struct {
		system, version string
		wantType        string
		wantURL         string
		wantLookup      bool
		wantChecksum    string
	}{
		{"x86_64-linux", "3.0.0", InstallArchive, "new", false, ChecksumGitHubRelease},
		{"aarch64-darwin", "3.0.0", InstallArchive, "new-mac", false, ChecksumGitHubRelease},
		// The first matching version override wins.
		{"x86_64-linux", "1.9.0", InstallBinary, "old", false, ChecksumNone},
		// The version override's own system override applies last.
		{"aarch64-darwin", "1.9.0", InstallBinary, "old-mac", false, ChecksumNone},
		// A lookup replaces url and checksum.
		{"x86_64-linux", "2.5.0", InstallArchive, "", true, ChecksumNone},
		// The definition's system override sets url, which replaces the lookup.
		{"aarch64-darwin", "2.5.0", InstallArchive, "new-mac", false, ChecksumNone},
	}
	for _, tt := range tests {
		t.Run(tt.system+"@"+tt.version, func(t *testing.T) {
			got := def.InstallFor(tt.system, tt.version)
			assert.Equal(t, tt.wantType, got.Type)
			assert.Equal(t, tt.wantURL, got.URL)
			assert.Equal(t, tt.wantLookup, got.Lookup != nil)
			assert.Equal(t, tt.wantChecksum, got.ChecksumType())
			assert.Nil(t, got.Overrides)
			assert.Nil(t, got.VersionOverrides)
		})
	}

	assert.NotEqual(t, def.InstallFingerprint("x86_64-linux", "1.9.0"), def.InstallFingerprint("x86_64-linux", "3.0.0"),
		"versions installed by different recipes have different fingerprints")
	assert.Equal(t, def.InstallFingerprint("x86_64-linux", "3.0.0"), def.InstallFingerprint("x86_64-linux", "3.1.0"))
}

func TestParseInstallErrors(t *testing.T) {
	tests := []struct {
		name    string
		install string
		wantErr string
	}{
		{
			name:    "url and lookup",
			install: `{"type": "archive", "url": "u", "lookup": {"url": "i", "select": ".x"}}`,
			wantErr: "install.url and .lookup are mutually exclusive",
		},
		{
			name:    "lookup without select",
			install: `{"type": "archive", "lookup": {"url": "i"}}`,
			wantErr: "install.lookup.url and .lookup.select are required",
		},
		{
			name:    "lookup with checksum",
			install: `{"type": "archive", "lookup": {"url": "i", "select": ".x"}, "checksum": {"type": "none"}}`,
			wantErr: "install.checksum is not used with .lookup",
		},
		{
			name:    "lookup select does not compile",
			install: `{"type": "archive", "lookup": {"url": "i", "select": ".[[["}}`,
			wantErr: "install.lookup.select: invalid jq expression",
		},
		{
			name:    "github_release checksum without repo",
			install: `{"type": "archive", "url": "u", "checksum": {"type": "github_release"}}`,
			wantErr: "install.checksum.repo must be of the form owner/name",
		},
		{
			name:    "bad version constraint",
			install: `{"type": "archive", "url": "u", "version_overrides": [{"versions": "<<1", "url": "v"}]}`,
			wantErr: "install.version_overrides[0].versions: invalid term",
		},
		{
			name:    "nested version overrides",
			install: `{"type": "archive", "url": "u", "version_overrides": [{"versions": "<1", "version_overrides": [{"versions": "<1"}]}]}`,
			wantErr: "install.version_overrides[0] may not contain version_overrides",
		},
		{
			name:    "version override is validated merged",
			install: `{"type": "archive", "url": "u", "version_overrides": [{"versions": "<1", "type": "script"}]}`,
			wantErr: "install.version_overrides[0].run is required for type script",
		},
		{
			name:    "version override system override for unknown platform",
			install: `{"type": "archive", "url": "u", "version_overrides": [{"versions": "<1", "overrides": {"aarch64-darwin": {"url": "m"}}}]}`,
			wantErr: `install.version_overrides[0].overrides: "aarch64-darwin" is not in platforms`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse([]byte(`{
			  "schema": 1, "name": "x", "platforms": ["x86_64-linux"],
			  "versions": {"type": "static", "versions": ["1"]},
			  "install": ` + tt.install + `
			}`))
			require.ErrorContains(t, err, tt.wantErr)
		})
	}
}

func TestUsesScriptsInVersionOverrides(t *testing.T) {
	def, err := Parse([]byte(minimalDef))
	require.NoError(t, err)
	def.Install.VersionOverrides = []*VersionOverride{{Versions: "<1", Install: Install{Type: InstallScript, Run: "true"}}}
	assert.True(t, def.UsesScripts())
}
