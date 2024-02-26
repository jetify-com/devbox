package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
)

func (m *Manager) ParseInclude(include string) (Includable, error) {
	if t, name, _ := strings.Cut(include, ":"); t == "plugin" {
		return devpkg.PackageFromStringWithDefaults(name, m.lockfile), nil
	}
	return parseReflike(include)
}
