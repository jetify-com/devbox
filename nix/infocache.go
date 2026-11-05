package nix

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"go.jetify.com/devbox/internal/xdg"
	"go.jetify.com/pkg/filecache"
)

// infoCacheTTL bounds how long a cached [Info] is trusted. The cache key and
// file stamps catch the common changes (a new Nix binary or an edited
// nix.conf), but not everything that can affect the output, such as a file
// pulled in by an include directive in nix.conf.
const infoCacheTTL = 24 * time.Hour

// infoCacheEnv are the environment variables that can change the output of
// "nix --version --debug" without changing the Nix binary or its config files.
var infoCacheEnv = []string{
	"HOME",
	"NIX_CONF_DIR",
	"NIX_CONFIG",
	"NIX_DATA_DIR",
	"NIX_STATE_DIR",
	"NIX_STORE_DIR",
	"NIX_USER_CONF_FILES",
	"XDG_CONFIG_DIRS",
	"XDG_CONFIG_HOME",
}

// cachedInfo is an [Info] stored on disk along with stamps of the files it
// was derived from.
type cachedInfo struct {
	Info  Info
	Files []fileStamp
}

// fileStamp records the state of a file at the time [Info] was cached. A
// missing file has a zero ModTime and Size so that creating it later
// invalidates the cache.
type fileStamp struct {
	Path    string
	ModTime time.Time
	Size    int64
}

func newFileStamp(path string) fileStamp {
	stamp := fileStamp{Path: path}
	if fi, err := os.Stat(path); err == nil {
		stamp.ModTime = fi.ModTime()
		stamp.Size = fi.Size()
	}
	return stamp
}

func (s fileStamp) changed() bool {
	current := newFileStamp(s.Path)
	return !current.ModTime.Equal(s.ModTime) || current.Size != s.Size
}

func newInfoCache() *filecache.Cache[cachedInfo] {
	return filecache.New("devbox/nix-info", filecache.WithCacheDir[cachedInfo](xdg.CacheSubpath("")))
}

// infoCacheKey returns the key for caching the output of cmd. It identifies the
// Nix binary by its resolved path (usually an immutable /nix/store path) and
// includes the command arguments and any environment variables that affect the
// output.
func infoCacheKey(cmd *Cmd) string {
	bin := cmd.Path
	if resolved, err := filepath.EvalSymlinks(bin); err == nil {
		bin = resolved
	}

	var key strings.Builder
	key.WriteString(bin)
	for _, arg := range cmd.Args.StringSlice()[1:] {
		key.WriteString("\x00")
		key.WriteString(arg)
	}
	for _, env := range infoCacheEnv {
		key.WriteString("\x00")
		key.WriteString(env)
		key.WriteString("=")
		key.WriteString(os.Getenv(env))
	}
	return key.String()
}

// infoFiles returns the files whose changes should invalidate a cached info:
// the Nix binary itself and the Nix config files that it reported.
func infoFiles(cmd *Cmd, info Info) []string {
	files := []string{cmd.Path}
	if info.SystemConfig != "" {
		files = append(files, info.SystemConfig)
	}
	for _, path := range info.UserConfigs {
		if path != "" {
			files = append(files, path)
		}
	}
	return files
}

// loadCachedInfo returns the cached info for key, or false if there's no
// cached info or if any of the files it was derived from have changed.
func loadCachedInfo(cache *filecache.Cache[cachedInfo], key string) (Info, bool) {
	cached, err := cache.Get(key)
	if err != nil || cached.Info.Version == "" {
		return Info{}, false
	}
	if slices.ContainsFunc(cached.Files, fileStamp.changed) {
		return Info{}, false
	}
	return cached.Info, true
}

// storeCachedInfo saves info to the cache, stamping the files in paths so that
// later changes to them invalidate it.
func storeCachedInfo(cache *filecache.Cache[cachedInfo], key string, info Info, paths []string) {
	cached := cachedInfo{Info: info, Files: make([]fileStamp, len(paths))}
	for i, path := range paths {
		cached.Files[i] = newFileStamp(path)
	}
	// Caching is best-effort. A failure only means running nix next time.
	_ = cache.Set(key, cached, infoCacheTTL)
}
