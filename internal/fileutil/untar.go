// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package fileutil

// TODO: publish as it's own shared package that other binaries can use.

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/mholt/archives"
)

func Untar(archive io.Reader, destPath string) error {
	_, err := os.Stat(destPath)
	if err != nil {
		return err
	}

	// We assume `tar.gz` since that's the only format we need for now.
	format := archives.CompressedArchive{
		Compression: archives.Gz{},
		Archival:    archives.Tar{},
	}

	// The handler will be called for each entry in the archive.
	handler := func(ctx context.Context, fromFile archives.FileInfo) error {
		// TODO: consider whether the path provided in the archive is a valid
		// relative path to begin with.
		rel := filepath.Clean(fromFile.NameInArchive)
		abs := filepath.Join(destPath, rel)

		mode := fromFile.Mode()

		// TODO: handle symlink case
		switch {
		case mode.IsRegular():
			return untarFile(fromFile, abs)
		case mode.IsDir():
			return os.MkdirAll(abs, 0o755)
		default:
			return fmt.Errorf("archive contained entry %s of unsupported file type %v", fromFile.Name(), mode)
		}
	}

	// Start extraction using our handler.
	return format.Extract(context.Background(), archive, handler)
}

func untarFile(fromFile archives.FileInfo, abs string) error {
	fromReader, err := fromFile.Open()
	if err != nil {
		return err
	}
	defer func() {
		closeErr := fromReader.Close()
		if closeErr != nil {
			log.Fatal(closeErr)
		}
	}()

	// We assume the directory exists because if the archive is constructed correctly
	// there should have been a directory entry already. If we want to be safer,
	// we could ensure the path exists before opening the file, although we should then
	// cache which directories we've already created for performance reasons.
	toWriter, err := os.OpenFile(abs, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fromFile.Mode().Perm())
	if err != nil {
		return err
	}
	numBytes, err := io.Copy(toWriter, fromReader)
	if closeErr := toWriter.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("error writing to %s: %v", abs, err)
	}
	if numBytes != fromFile.Size() {
		return fmt.Errorf("only wrote %d bytes to %s; expected %d", numBytes, abs, fromFile.Size())
	}
	modTime := fromFile.ModTime()
	if !modTime.IsZero() {
		if err := os.Chtimes(abs, modTime, modTime); err != nil {
			return err
		}
	}
	return nil
}
