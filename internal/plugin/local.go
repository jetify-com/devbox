package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/nix/flake"
)

type LocalPlugin struct {
	ref       flake.Ref
	name      string
	pluginDir string
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)

func newLocalPlugin(ref flake.Ref, pluginDir string) (*LocalPlugin, error) {
	plugin := &LocalPlugin{ref: ref, pluginDir: pluginDir}
	content, err := plugin.Fetch()
	if err != nil {
		return nil, err
	}
	m := map[string]any{}
	if err := json.Unmarshal(content, &m); err != nil {
		return nil, err
	}
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return nil,
			usererr.New("plugin %s is missing a required field 'name'", plugin.Path())
	}
	if !nameRegex.MatchString(name) {
		return nil, usererr.New(
			"plugin %s has an invalid name %q. Name must match %s",
			plugin.Path(), name, nameRegex,
		)
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
