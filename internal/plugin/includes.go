package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/nix"
)

func (m *Manager) parseInclude(include string) (*nix.Input, error) {
	includeType, name, _ := strings.Cut(include, ":")
	if includeType != "plugin" {
		return nil, usererr.New("unknown include type %q", includeType)
	} else if name == "" {
		return nil, usererr.New("include name is required")
	}
	return nix.InputFromString(name, m.ProjectDir(), m.lockfile), nil
}
