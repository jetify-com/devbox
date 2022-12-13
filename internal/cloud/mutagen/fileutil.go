package mutagen

import (
	"os"
)

// TODO: publish as it's own shared package that other binaries
// can use.

// IsDir returns true if the path exists *and* it is pointing to a directory.
//
// This function will traverse symbolic links to query information about the
// destination file.
//
// This is a convenience function that coerces errors to false. If it cannot
// read the path for any reason (including a permission error, or a broken
// symbolic link) it returns false.
func IsDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// IsFile returns true if the path exists *and* it is pointing to a regular file.
//
// This function will traverse symbolic links to query information about the
// destination file.
//
// This is a convenience function that coerces errors to false. If it cannot
// read the path for any reason (including a permission error, or a broken
// symbolic link) it returns false.
func IsFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.Mode().IsRegular()
}

// IsSymlink returns true if the path exists *and* it is a symlink.
//
// It does *not* traverse symbolic links, and returns true even if the symlink
// is broken.
//
// This is a convenience function that coerces errors to false. If it cannot
// read the path for any reason (including a permission error) it returns false.
func IsSymlink(path string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return (info.Mode().Type() & os.ModeSymlink) == os.ModeSymlink
}

func ExistsOrErr(path string) error {
	_, err := os.Stat(path)
	return err
}
