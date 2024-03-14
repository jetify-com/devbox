package plugin

import (
	"os"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/nix/flake"
)

type LocalPlugin struct {
	ref       flake.Ref
	name      string
	pluginDir string
}

func newLocalPlugin(ref flake.Ref, pluginDir string) (*LocalPlugin, error) {
	plugin := &LocalPlugin{ref: ref, pluginDir: pluginDir}
	name, err := getPluginNameFromContent(plugin)
	if err != nil {
		return nil, err
	}
	plugin.name = name
	return plugin, nil
}

func (l *LocalPlugin) Fetch() ([]byte, error) {
	return os.ReadFile(addFilenameIfMissing(l.Path()))
}

func (l *LocalPlugin) CanonicalName() string {
	return l.name
}

func (l *LocalPlugin) IsLocal() bool {
	return true
}

func (l *LocalPlugin) Hash() string {
	h, _ := cachehash.Bytes([]byte(filepath.Clean(l.Path())))
	return h
}

func (l *LocalPlugin) FileContent(subpath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(filepath.Dir(l.Path()), subpath))
}

func (l *LocalPlugin) LockfileKey() string {
	return l.ref.String()
}

func (l *LocalPlugin) Path() string {
	path := l.ref.Path
	if !strings.HasSuffix(path, pluginConfigName) {
		path = filepath.Join(path, pluginConfigName)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.pluginDir, path)
}

func addFilenameIfMissing(s string) string {
	if strings.HasSuffix(s, pluginConfigName) {
		return s
	}
	return filepath.Join(s, pluginConfigName)
}
