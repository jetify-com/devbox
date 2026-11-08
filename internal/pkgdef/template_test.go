// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateVarsDefaults(t *testing.T) {
	def := &Definition{}
	vars, err := def.TemplateVars("aarch64-darwin", "1.22.3")
	require.NoError(t, err)
	assert.Equal(t, Vars{Version: "1.22.3", System: "aarch64-darwin", OS: "darwin", Arch: "arm64"}, vars)

	vars, err = def.TemplateVars("x86_64-linux", "1")
	require.NoError(t, err)
	assert.Equal(t, "linux", vars.OS)
	assert.Equal(t, "amd64", vars.Arch)

	_, err = def.TemplateVars("windows", "1")
	require.Error(t, err)
}

func TestTemplateVarsMaps(t *testing.T) {
	def := &Definition{
		OSMap:   map[string]string{"darwin": "macos"},
		ArchMap: map[string]string{"x86_64": "x64", "aarch64": "aarch64"},
	}
	vars, err := def.TemplateVars("aarch64-darwin", "1")
	require.NoError(t, err)
	assert.Equal(t, "macos", vars.OS)
	assert.Equal(t, "aarch64", vars.Arch)

	vars, err = def.TemplateVars("x86_64-linux", "1")
	require.NoError(t, err)
	assert.Equal(t, "linux", vars.OS, "unmapped OS keeps the default")
	assert.Equal(t, "x64", vars.Arch)
}

func TestRender(t *testing.T) {
	vars := Vars{Version: "1.22.3", OS: "darwin", Arch: "arm64"}

	got, err := Render("https://go.dev/dl/go{{.Version}}.{{.OS}}-{{.Arch}}.tar.gz", vars)
	require.NoError(t, err)
	assert.Equal(t, "https://go.dev/dl/go1.22.3.darwin-arm64.tar.gz", got)

	got, err = Render(`{{ .Version | trimPrefix "1." }}-{{ upper .OS }}-{{ replace "arm" "ARM" .Arch }}-{{ trimSuffix ".3" .Version }}`, vars)
	require.NoError(t, err)
	assert.Equal(t, "22.3-DARWIN-ARM64-1.22", got)

	got, err = Render("no templates here", vars)
	require.NoError(t, err)
	assert.Equal(t, "no templates here", got)

	_, err = Render("{{.Nope}}", vars)
	require.Error(t, err)

	_, err = Render("{{.Version", vars)
	require.Error(t, err)
}

func TestWithURL(t *testing.T) {
	vars := Vars{}.withURL("https://go.dev/dl/go1.22.3.darwin-arm64.tar.gz?x=1")
	assert.Equal(t, "https://go.dev/dl/go1.22.3.darwin-arm64.tar.gz?x=1", vars.URL)
	assert.Equal(t, "go1.22.3.darwin-arm64.tar.gz", vars.Filename)

	assert.Equal(t, "hello.tgz", filenameFromURL("file:///tmp/hello.tgz"))
}
