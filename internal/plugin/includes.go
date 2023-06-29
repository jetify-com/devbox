package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
)

type includable interface {
	CanonicalName() string
}

func (m *Manager) parseInclude(include string) (includable, error) {
	includeType, name, _ := strings.Cut(include, ":")
	if name == "" {
		return nil, usererr.New("include name is required")
	} else if includeType == "plugin" {
		return nix.PackageFromString(name, m.lockfile), nil
	} else if includeType == "path" {
		absPath := filepath.Join(m.ProjectDir(), name)
		return newLocalPlugin(absPath)
	}
	return nil, usererr.New("unknown include type %q", includeType)
}

type localPlugin struct {
	name string
	path string
}

func newLocalPlugin(path string) (*localPlugin, error) {
	content, err := os.ReadFile(path)
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
			usererr.New("plugin %s is missing a required field 'name'", path)
	}
	return &localPlugin{
		name: name,
		path: path,
	}, nil
}

func (l *localPlugin) CanonicalName() string {
	return l.name
}

func (l *localPlugin) IsLocal() bool {
	return true
}

func (l *localPlugin) contentPath(subpath string) string {
	return filepath.Join(filepath.Dir(l.path), subpath)
}
