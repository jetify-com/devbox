package patchpkg

import (
	"fmt"
	"io"
	"io/fs"
	"iter"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
)

// maxFileSize limits the amount of data to load from a file when
// searching.
const maxFileSize = 1 << 30 // 1 GiB

// reRemovedRefs matches a removed Nix store path where the hash is
// overwritten with e's (making it an invalid nix hash).
var reRemovedRefs = regexp.MustCompile(`e{32}-[^$"'{}/[\] \t\r\n]+`)

// fileSlice is a slice of data within a file.
type fileSlice struct {
	path   string
	data   []byte
	offset int64
}

func (f fileSlice) String() string {
	return fmt.Sprintf("%s@%d: %s", f.path, f.offset, f.data)
}

// searchFile searches a single file for a regular expression. It limits the
// search to the first [maxFileSize] bytes of the file to avoid consuming too
// much memory.
func searchFile(fsys fs.FS, path string, re *regexp.Regexp) ([]fileSlice, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	r := &io.LimitedReader{R: f, N: maxFileSize}
	data, err := io.ReadAll(r)
	if err != nil {
		return nil, err
	}

	locs := re.FindAllIndex(data, -1)
	if len(locs) == 0 {
		return nil, nil
	}

	matches := make([]fileSlice, len(locs))
	for i := range locs {
		start, end := locs[i][0], locs[i][1]
		matches[i] = fileSlice{
			path:   path,
			data:   data[start:end],
			offset: int64(start),
		}
	}
	return matches, nil
}

var envValues = sync.OnceValue(func() []string {
	env := os.Environ()
	values := make([]string, len(env))
	for i := range env {
		_, values[i], _ = strings.Cut(env[i], "=")
	}
	return values
})

func searchEnv(re *regexp.Regexp) string {
	for _, env := range envValues() {
		match := re.FindString(env)
		if match != "" {
			return match
		}
	}
	return ""
}

// SystemCUDALibraries returns an iterator over the system CUDA library paths.
// It yields them in priority order, where the first path is the most likely to
// be the correct version.
var SystemCUDALibraries iter.Seq[string] = func(yield func(string) bool) {
	// Quick overview of Unix-like shared object versioning.
	//
	// Libraries have 3 different names (using libcuda as an example):
	//
	//  1. libcuda.so - the "linker name". Typically a symlink pointing to
	//     the soname. The compiler looks for this name.
	//  2. libcuda.so.1 - the "soname". Typically a symlink pointing to the
	//     real name. The dynamic linker looks for this name.
	//  3. libcuda.so.550.107.02 - the "real name". The actual ELF shared
	//     library. Usually never referred to directly because that would
	//     make versioning hard.
	//
	// Because we don't know what version of CUDA the user's program
	// actually needs, we're going to try to find linker names (libcuda.so)
	// and trust that the system is pointing it to the correct version.
	// We'll fall back to sonames (libcuda.so.1) that we find if none of the
	// linker names work.

	// Common direct paths to try first.
	linkerNames := []string{
		"/usr/lib/x86_64-linux-gnu/libcuda.so", // Debian
		"/usr/lib64/libcuda.so",                // Red Hat
		"/usr/lib/libcuda.so",
	}
	for _, path := range linkerNames {
		// Return what the link is pointing to because the dynamic
		// linker will want libcuda.so.1, not libcuda.so.
		soname, err := os.Readlink(path)
		if err != nil {
			continue
		}
		if filepath.IsLocal(soname) {
			soname = filepath.Join(filepath.Dir(path), soname)
		}
		if !yield(soname) {
			return
		}
	}

	// Directories to recursively search.
	prefixes := []string{
		"/usr/lib",
		"/usr/lib64",
		"/lib",
		"/lib64",
		"/usr/local/lib",
		"/usr/local/lib64",
		"/opt/cuda",
		"/opt/nvidia",
		"/usr/local/cuda",
		"/usr/local/nvidia",
	}
	sonameRegex := regexp.MustCompile(`^libcuda\.so\.\d+$`)
	var sonames []string
	for _, path := range prefixes {
		_ = filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.Name() == "libcuda.so" && isSymlink(entry.Type()) {
				soname, err := os.Readlink(path)
				if err != nil {
					return nil
				}
				if filepath.IsLocal(soname) {
					soname = filepath.Join(filepath.Dir(path), soname)
				}
				if !yield(soname) {
					return filepath.SkipAll
				}
			}

			// Save potential soname matches for later after we've
			// exhausted all the potential linker names.
			if sonameRegex.MatchString(entry.Name()) {
				sonames = append(sonames, entry.Name())
			}
			return nil
		})
	}

	// We didn't find any symlinks named libcuda.so. Fall back to trying any
	// sonames (e.g., libcuda.so.1) that we found.
	for _, path := range sonames {
		if !yield(path) {
			return
		}
	}
}
