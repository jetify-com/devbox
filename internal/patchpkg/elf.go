package patchpkg

import (
	"debug/elf"
	"errors"
	"fmt"
	"io"
	"iter"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

var (
	// SystemLibSearchPaths match the system library paths for common Linux
	// distributions.
	SystemLibSearchPaths = []string{
		"/lib*/*-linux-gnu", // Debian
		"/lib*",             // Red Hat
		"/var/lib*/*/lib*",  // Docker
	}

	// EnvLibrarySearchPath matches the paths in the LIBRARY_PATH
	// environment variable.
	EnvLibrarySearchPath = filepath.SplitList(os.Getenv("LIBRARY_PATH"))

	// EnvLDLibrarySearchPath matches the paths in the LD_LIBRARY_PATH
	// environment variable.
	EnvLDLibrarySearchPath = filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))

	// CUDALibSearchPaths match the common installation directories for CUDA
	// libraries.
	CUDALibSearchPaths = []string{
		// Common non-package manager locations.
		"/opt/cuda/lib*",
		"/opt/nvidia/lib*",
		"/usr/local/cuda/lib*",
		"/usr/local/nvidia/lib*",

		// Unlikely, but might as well try.
		"/lib*/nvidia",
		"/lib*/cuda",
		"/usr/lib*/nvidia",
		"/usr/lib*/cuda",
		"/usr/local/lib*",
		"/usr/local/lib*/nvidia",
		"/usr/local/lib*/cuda",
	}
)

// SharedLibrary describes an ELF shared library (object).
//
// Note that the various name fields document the common naming and versioning
// conventions, but it is possible for a library to deviate from them.
//
// For an introduction to Linux shared libraries, see
// https://tldp.org/HOWTO/Program-Library-HOWTO/shared-libraries.html
type SharedLibrary struct {
	*os.File

	// LinkerName is the soname without any version suffix (libfoo.so). It
	// is typically a symlink pointing to Soname. The build-time linker
	// looks for this name by default.
	LinkerName string

	// Soname is the shared object name from the library's DT_SONAME field.
	// It usually includes a version number suffix (libfoo.so.1). Other ELF
	// binaries that depend on this library typically specify this name in
	// the DT_NEEDED field.
	Soname string

	// RealName is the absolute path to the file that actually contains the
	// library code. It is typically the soname plus a minor version and
	// release number (libfoo.so.1.0.0).
	RealName string
}

// OpenSharedLibrary opens a shared library file. Unlike with ld, name must be
// an exact path. To search for a library in the usual locations, use
// [FindSharedLibrary] instead.
func OpenSharedLibrary(name string) (SharedLibrary, error) {
	lib := SharedLibrary{}
	var err error
	lib.File, err = os.Open(name)
	if err != nil {
		return lib, err
	}

	dir, file := filepath.Split(name)
	i := strings.Index(file, ".so")
	if i != -1 {
		lib.LinkerName = dir + file[:i+3]
	}

	elfFile, err := elf.NewFile(lib)
	if err == nil {
		soname, _ := elfFile.DynString(elf.DT_SONAME)
		if len(soname) != 0 {
			lib.Soname = soname[0]
		}
	}

	real, err := filepath.EvalSymlinks(name)
	if err == nil {
		lib.RealName, _ = filepath.Abs(real)
	}
	return lib, nil
}

// FindSharedLibrary searches the directories in searchPath for a shared
// library. It yields any libraries in the search path directories that have
// name as a prefix. For example, "libcuda.so" will match "libcuda.so",
// "libcuda.so.1", and "libcuda.so.550.107.02". The underlying file is only
// valid for a single iteration, after which it is closed.
//
// The search path may contain [filepath.Glob] patterns. See
// [SystemLibSearchPaths] for some predefined search paths. If name is an
// absolute path, then FindSharedLibrary opens it directly and doesn't perform
// any searching.
func FindSharedLibrary(name string, searchPath ...string) iter.Seq[SharedLibrary] {
	return func(yield func(SharedLibrary) bool) {
		if filepath.IsAbs(name) {
			lib, err := OpenSharedLibrary(name)
			if err == nil {
				yield(lib)
			}
			return
		}

		if libPath := os.Getenv("LD_LIBRARY_PATH"); libPath != "" {
			searchPath = append(searchPath, filepath.SplitList(os.Getenv("LD_LIBRARY_PATH"))...)
		}
		if libPath := os.Getenv("LIBRARY_PATH"); libPath != "" {
			searchPath = append(searchPath, filepath.SplitList(libPath)...)
		}
		searchPath = append(searchPath,
			"/lib*/*-linux-gnu", // Debian
			"/lib*",             // Red Hat
		)

		suffix := globEscape(name) + "*"
		patterns := make([]string, len(searchPath))
		for i := range searchPath {
			patterns[i] = filepath.Join(searchPath[i], suffix)
		}
		for match := range searchGlobs(patterns) {
			lib, err := OpenSharedLibrary(match)
			if err != nil {
				continue
			}
			ok := yield(lib)
			_ = lib.Close()
			if !ok {
				return
			}
		}
	}
}

// CopyAndLink copies the shared library to dir and creates the LinkerName and
// Soname symlinks for it. It creates dir if it doesn't already exist.
func (lib SharedLibrary) CopyAndLink(dir string) error {
	err := os.MkdirAll(dir, 0o755)
	if err != nil {
		return err
	}

	dstPath := filepath.Join(dir, filepath.Base(lib.RealName))
	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, 0o666)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, lib)
	if err != nil {
		return err
	}
	err = dst.Close()
	if err != nil {
		return err
	}

	sonameLink := filepath.Join(dir, filepath.Base(lib.Soname))
	var sonameErr error
	if lib.Soname != "" {
		// Symlink must be relative.
		sonameErr = os.Symlink(filepath.Base(lib.RealName), sonameLink)
	}

	linkerNameLink := filepath.Join(dir, filepath.Base(lib.LinkerName))
	var linkerNameErr error
	if lib.LinkerName != "" {
		// Symlink must be relative.
		if sonameErr == nil {
			linkerNameErr = os.Symlink(filepath.Base(sonameLink), linkerNameLink)
		} else {
			linkerNameErr = os.Symlink(filepath.Base(dstPath), linkerNameLink)
		}
	}

	err = errors.Join(sonameErr, linkerNameErr)
	if err != nil {
		return fmt.Errorf("patchpkg: create symlinks for shared library: %w", err)
	}
	return nil
}

func (lib SharedLibrary) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("path", lib.Name()),
		slog.String("linkername", lib.LinkerName),
		slog.String("soname", lib.Soname),
		slog.String("realname", lib.RealName),
	)
}
