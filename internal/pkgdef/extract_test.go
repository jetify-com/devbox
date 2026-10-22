// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type tarEntry struct {
	name     string
	content  string
	linkTo   string
	dir      bool
	mode     int64
	typeflag byte
}

func writeTar(t *testing.T, path string, entries []tarEntry) {
	t.Helper()
	var buf bytes.Buffer
	gzw := gzip.NewWriter(&buf)
	tarw := tar.NewWriter(gzw)
	for _, entry := range entries {
		hdr := &tar.Header{Name: entry.name, Mode: entry.mode}
		switch {
		case entry.dir:
			hdr.Typeflag = tar.TypeDir
		case entry.linkTo != "":
			hdr.Typeflag = tar.TypeSymlink
			hdr.Linkname = entry.linkTo
		case entry.typeflag != 0:
			hdr.Typeflag = entry.typeflag
		default:
			hdr.Typeflag = tar.TypeReg
			hdr.Size = int64(len(entry.content))
		}
		if hdr.Mode == 0 {
			hdr.Mode = 0o644
		}
		require.NoError(t, tarw.WriteHeader(hdr))
		if hdr.Typeflag == tar.TypeReg {
			_, err := tarw.Write([]byte(entry.content))
			require.NoError(t, err)
		}
	}
	require.NoError(t, tarw.Close())
	require.NoError(t, gzw.Close())
	require.NoError(t, os.WriteFile(path, buf.Bytes(), 0o644))
}

func TestExtractTarWithSymlinksAndStrip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "node.tar.gz")
	writeTar(t, archivePath, []tarEntry{
		{name: "node-v20/", dir: true, mode: 0o755},
		{name: "node-v20/bin/", dir: true, mode: 0o755},
		{name: "node-v20/bin/node", content: "#!/bin/sh\necho node\n", mode: 0o755},
		{name: "node-v20/lib/npm/npm-cli.js", content: "js\n"},
		{name: "node-v20/bin/npm", linkTo: "../lib/npm/npm-cli.js", mode: 0o777},
		{name: "node-v20/dev/null", typeflag: tar.TypeChar},
	})

	dest := filepath.Join(dir, "out")
	strip := 1
	require.NoError(t, extractArchive(t.Context(), archivePath, dest, &strip))

	node, err := os.Stat(filepath.Join(dest, "bin", "node"))
	require.NoError(t, err)
	assert.NotZero(t, node.Mode()&0o111, "executable bit is preserved")

	target, err := os.Readlink(filepath.Join(dest, "bin", "npm"))
	require.NoError(t, err)
	assert.Equal(t, filepath.FromSlash("../lib/npm/npm-cli.js"), target)
	content, err := os.ReadFile(filepath.Join(dest, "bin", "npm"))
	require.NoError(t, err, "symlink resolves inside the install")
	assert.Equal(t, "js\n", string(content))

	_, err = os.Stat(filepath.Join(dest, "node-v20"))
	assert.True(t, os.IsNotExist(err), "the stripped directory is gone")
	_, err = os.Lstat(filepath.Join(dest, "dev", "null"))
	assert.True(t, os.IsNotExist(err), "device entries are skipped")
}

func TestExtractZipNoStrip(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "tool.zip")
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("bin/tool")
	require.NoError(t, err)
	_, err = w.Write([]byte("tool\n"))
	require.NoError(t, err)
	require.NoError(t, zw.Close())
	require.NoError(t, os.WriteFile(archivePath, buf.Bytes(), 0o644))

	dest := filepath.Join(dir, "out")
	require.NoError(t, extractArchive(t.Context(), archivePath, dest, nil))
	content, err := os.ReadFile(filepath.Join(dest, "bin", "tool"))
	require.NoError(t, err)
	assert.Equal(t, "tool\n", string(content))
}

func TestExtractRejectsEscapes(t *testing.T) {
	dir := t.TempDir()

	// A symlink pointing outside the destination is refused.
	archivePath := filepath.Join(dir, "escape-link.tar.gz")
	writeTar(t, archivePath, []tarEntry{
		{name: "pkg/bin/evil", linkTo: "../../../../etc/passwd", mode: 0o777},
	})
	err := extractArchive(t.Context(), archivePath, filepath.Join(dir, "out1"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "escapes the destination")

	// So is an absolute symlink target.
	archivePath = filepath.Join(dir, "abs-link.tar.gz")
	writeTar(t, archivePath, []tarEntry{
		{name: "pkg/bin/evil", linkTo: "/etc/passwd", mode: 0o777},
	})
	err = extractArchive(t.Context(), archivePath, filepath.Join(dir, "out2"), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported symlink target")

	// A file that escapes via ".." lands inside the destination instead:
	// the path is anchored and cleaned before use.
	archivePath = filepath.Join(dir, "dotdot.tar.gz")
	writeTar(t, archivePath, []tarEntry{
		{name: "../../outside.txt", content: "x"},
	})
	dest := filepath.Join(dir, "out3")
	require.NoError(t, extractArchive(t.Context(), archivePath, dest, nil))
	_, err = os.Stat(filepath.Join(dest, "outside.txt"))
	require.NoError(t, err)
	_, err = os.Stat(filepath.Join(dir, "outside.txt"))
	assert.True(t, os.IsNotExist(err))

	// A regular file written through a planted symlink replaces the link
	// rather than following it.
	outside := filepath.Join(dir, "victim.txt")
	require.NoError(t, os.WriteFile(outside, []byte("untouched"), 0o644))
	archivePath = filepath.Join(dir, "through-link.tar.gz")
	writeTar(t, archivePath, []tarEntry{
		{name: "pkg/link", linkTo: "sub/real", mode: 0o777},
		{name: "pkg/link", content: "overwritten"},
	})
	dest = filepath.Join(dir, "out4")
	require.NoError(t, extractArchive(t.Context(), archivePath, dest, nil))
	info, err := os.Lstat(filepath.Join(dest, "pkg", "link"))
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular())
	content, err := os.ReadFile(outside)
	require.NoError(t, err)
	assert.Equal(t, "untouched", string(content))
}

func TestStripPath(t *testing.T) {
	tests := []struct {
		name  string
		strip int
		want  string
		ok    bool
	}{
		{"go/bin/go", 1, filepath.FromSlash("bin/go"), true},
		{"./go/bin/go", 1, filepath.FromSlash("bin/go"), true},
		{"go/", 1, "", false},
		{"go", 1, "", false},
		{"bin/go", 0, filepath.FromSlash("bin/go"), true},
		{"../../x", 0, "x", true},
		{"./", 0, "", false},
		{"a/../b/c", 1, "c", true},
	}
	for _, test := range tests {
		got, ok := stripPath(test.name, test.strip)
		assert.Equal(t, test.ok, ok, test.name)
		assert.Equal(t, test.want, got, test.name)
	}
}

func TestExtractUnknownFormat(t *testing.T) {
	dir := t.TempDir()
	archivePath := filepath.Join(dir, "notes.txt")
	require.NoError(t, os.WriteFile(archivePath, []byte("just text"), 0o644))
	err := extractArchive(t.Context(), archivePath, filepath.Join(dir, "out"), nil)
	require.Error(t, err)
}
