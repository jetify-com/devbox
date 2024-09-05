package patchpkg

import (
	"fmt"
	"io"
	"io/fs"
	"os"
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
