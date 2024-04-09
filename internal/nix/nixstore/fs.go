// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixstore

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// readLinkFS is an [os.DirFS] that supports reading symlinks. It satisfies
// the interface discussed in the [accepted Go proposal for fs.ReadLinkFS].
// If differs from the proposed implementation in that it allows absolute
// symlinks by translating them to relative paths.
//
// [accepted Go proposal for fs.ReadLinkFS]: https://github.com/golang/go/issues/49580
type readLinkFS struct {
	fs.FS
	dir string
}

func newReadLinkFS(dir string) fs.FS {
	return &readLinkFS{FS: os.DirFS(dir), dir: dir}
}

func (fsys *readLinkFS) ReadLink(name string) (string, error) {
	osName := filepath.Join(fsys.dir, filepath.FromSlash(name))
	dst, err := os.Readlink(osName)
	if err != nil {
		return "", err
	}
	if !filepath.IsAbs(dst) {
		dst = filepath.Join(filepath.Dir(osName), dst)
	}
	if filepath.IsAbs(dst) {
		dst, err = filepath.Rel(fsys.dir, dst)
		if err != nil {
			return "", fmt.Errorf("%s evaluates to a path outside of the root", name)
		}
	}
	if !filepath.IsLocal(dst) {
		return "", fmt.Errorf("%s evaluates to a path outside of the root", name)
	}
	return dst, nil
}

// readLink returns the destination of a symbolic link. If the file system
// doesn't implement ReadLink, then it returns an error. It matches the
// interface discussed in the [accepted Go proposal for fs.ReadLink].
//
// [accepted Go proposal for fs.ReadLink]: https://github.com/golang/go/issues/49580
func readLink(fsys fs.FS, name string) (string, error) {
	rlFS, ok := fsys.(interface{ ReadLink(string) (string, error) })
	if !ok {
		return "", &fs.PathError{
			Op:   "readlink",
			Path: name,
			Err:  errors.New("not implemented"),
		}
	}
	return rlFS.ReadLink(name)
}

// readDirUnsorted acts identically to [fs.ReadDir] except that it skips
// sorting the directory entries when possible to save some time.
func readDirUnsorted(fsys fs.FS, path string) ([]fs.DirEntry, error) {
	if fsys, ok := fsys.(fs.ReadDirFS); ok {
		return fsys.ReadDir(path)
	}
	f, err := fsys.Open(".")
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dir, ok := f.(fs.ReadDirFile)
	if !ok {
		return nil, &fs.PathError{
			Op:   "readdir",
			Path: path,
			Err:  errors.New("not implemented"),
		}
	}
	return dir.ReadDir(-1)
}
