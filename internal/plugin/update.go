package plugin

import (
	"os"
	"path/filepath"
)

func Update() error {
	// TODO: Implement in filecache
	// githubCache.Clear()
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(cacheDir, "devbox/plugin/github"))
}
