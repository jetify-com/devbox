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

type localPlugin struct {
	ref        flake.Ref
	name       string
	projectDir string
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)

func newLocalPlugin(ref flake.Ref, projectDir string) (*localPlugin, error) {
	plugin := &localPlugin{ref: ref, projectDir: projectDir}
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

func (l *localPlugin) Fetch() ([]byte, error) {
	return os.ReadFile(addFilenameIfMissing(l.Path()))
}

func (l *localPlugin) CanonicalName() string {
	return l.name
}

func (l *localPlugin) IsLocal() bool {
	return true
}

func (l *localPlugin) Hash() string {
	h, _ := cachehash.Bytes([]byte(l.Path()))
	return h
}

func (l *localPlugin) FileContent(subpath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(filepath.Dir(l.Path()), subpath))
}

func (l *localPlugin) LockfileKey() string {
	return l.ref.String()
}

func (l *localPlugin) Path() string {
	path := l.ref.Path
	if !strings.HasSuffix(path, pluginConfigName) {
		path = filepath.Join(path, pluginConfigName)
	}
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(l.projectDir, path)
}

func addFilenameIfMissing(s string) string {
	if strings.HasSuffix(s, pluginConfigName) {
		return s
	}
	return filepath.Join(s, pluginConfigName)
}
