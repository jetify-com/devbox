// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTarGz(t *testing.T, files map[string]string) *bytes.Buffer {
	t.Helper()
	buf := &bytes.Buffer{}
	gzipWriter := gzip.NewWriter(buf)
	tarWriter := tar.NewWriter(gzipWriter)
	for name, content := range files {
		require.NoError(t, tarWriter.WriteHeader(&tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}))
		_, err := tarWriter.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, tarWriter.Close())
	require.NoError(t, gzipWriter.Close())
	return buf
}

func TestUntar(t *testing.T) {
	dest := t.TempDir()
	err := Untar(makeTarGz(t, map[string]string{"hello.txt": "hi"}), dest)
	require.NoError(t, err)

	got, err := os.ReadFile(filepath.Join(dest, "hello.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hi", string(got))
}

func TestUntarRejectsPathTraversal(t *testing.T) {
	root := t.TempDir()
	dest := filepath.Join(root, "dest")
	require.NoError(t, os.Mkdir(dest, 0o755))

	err := Untar(makeTarGz(t, map[string]string{"../evil.txt": "pwned"}), dest)
	require.Error(t, err)
	assert.NoFileExists(t, filepath.Join(root, "evil.txt"))
}
