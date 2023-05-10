package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

func parseInclude(include string) (string, error) {
	includeType, name, _ := strings.Cut(include, ":")
	if includeType != "plugin" {
		return "", usererr.New("unknown include type %q", includeType)
	} else if name == "" {
		return "", usererr.New("include name is required")
	}
	return name, nil
}
