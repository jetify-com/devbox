package envpath

import (
	"path/filepath"
	"strings"
)

// JoinPathLists joins and cleans PATH-style strings of
// [os.ListSeparator] delimited paths. To clean a path list, it splits it and
// does the following for each element:
//
//  1. Applies [filepath.Clean].
//  2. Removes the path if it's relative (must begin with '/' and not be '.').
//  3. Removes the path if it's a duplicate.
func JoinPathLists(pathLists ...string) string {
	if len(pathLists) == 0 {
		return ""
	}

	seen := make(map[string]bool)
	var cleaned []string
	for _, path := range pathLists {
		for _, path := range filepath.SplitList(path) {
			path = filepath.Clean(path)
			if path == "." || path[0] != '/' {
				// Remove empty paths and don't allow relative
				// paths for security reasons.
				continue
			}
			if !seen[path] {
				cleaned = append(cleaned, path)
			}
			seen[path] = true
		}
	}
	return strings.Join(cleaned, string(filepath.ListSeparator))
}
