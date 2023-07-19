package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/devpkg"
)

type Includable interface {
	CanonicalName() string
	Hash() string
	FileContent(subpath string) ([]byte, error)
}

func (m *Manager) ParseInclude(include string) (Includable, error) {
	includeType, name, _ := strings.Cut(include, ":")
	if name == "" {
		return nil, usererr.New("include name is required")
	} else if includeType == "plugin" {
		return devpkg.PackageFromString(name, m.lockfile), nil
	} else if includeType == "path" {
		absPath := filepath.Join(m.ProjectDir(), name)
		return newLocalPlugin(absPath)
	} else if includeType == "github" {
		return newGithubPlugin(name)
	}
	return nil, usererr.New("unknown include type %q", includeType)
}

type localPlugin struct {
	name string
	path string
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)

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
	if !nameRegex.MatchString(name) {
		return nil, usererr.New(
			"plugin %s has an invalid name %q. Name must match %s",
			path, name, nameRegex,
		)
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

func (l *localPlugin) Hash() string {
	h, _ := cuecfg.FileHash(l.path)
	return h
}

func (l *localPlugin) FileContent(subpath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(filepath.Dir(l.path), subpath))
}
