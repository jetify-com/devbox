package plugin

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
)

type localPlugin struct {
	ref  RefLike
	name string
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)

func newLocalPlugin(ref RefLike) (*localPlugin, error) {
	plugin := &localPlugin{ref: ref}
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
			usererr.New("plugin %s is missing a required field 'name'", plugin.ref.Path)
	}
	if !nameRegex.MatchString(name) {
		return nil, usererr.New(
			"plugin %s has an invalid name %q. Name must match %s",
			plugin.ref.Path, name, nameRegex,
		)
	}
	plugin.name = name
	return plugin, nil
}

func (l *localPlugin) Fetch() ([]byte, error) {
	return os.ReadFile(l.ref.withFilename(l.ref.Path))
}

func (l *localPlugin) CanonicalName() string {
	return l.name
}

func (l *localPlugin) IsLocal() bool {
	return true
}

func (l *localPlugin) Hash() string {
	h, _ := cachehash.Bytes([]byte(l.ref.Path))
	return h
}

func (l *localPlugin) FileContent(subpath string) ([]byte, error) {
	return os.ReadFile(filepath.Join(filepath.Dir(l.ref.Path), subpath))
}
