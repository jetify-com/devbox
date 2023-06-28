package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
)

func (m *Manager) parseInclude(include string) (*nix.Package, error) {
	includeType, name, _ := strings.Cut(include, ":")
	if name == "" {
		return nil, usererr.New("include name is required")
	} else if includeType == "plugin" {
		return nix.PackageFromString(name, m.lockfile), nil
	} else if includeType == "path" {
		return nix.PackageFromString(include, m.lockfile), nil
	}
	return nil, usererr.New("unknown include type %q", includeType)
}
