package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
)

func (m *Manager) ParseInclude(include string) (Includable, error) {
	if t, name, _ := strings.Cut(include, ":"); t == "plugin" {
		return devpkg.PackageFromStringWithDefaults(name, m.lockfile), nil
	}
	return parseReflike(include)
}

func LoadConfigFromInclude(include, projectDir string) (*Config, error) {
	var includable Includable
	var err error
	if t, name, _ := strings.Cut(include, ":"); t == "plugin" {
		includable = devpkg.PackageFromStringWithDefaults(
			name,
			&lock.DummyLocker{ProjectDirVal: projectDir},
		)
	} else {
		includable, err = parseReflike(include)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, projectDir)
}
