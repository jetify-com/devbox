package plugin

import (
	"strings"

	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
)

func LoadConfigFromInclude(include string, lockfile *lock.File, workingDir string) (*Config, error) {
	var includable Includable
	var err error
	if t, name, _ := strings.Cut(include, ":"); t == "plugin" {
		includable = devpkg.PackageFromStringWithDefaults(
			name,
			lockfile,
		)
	} else {
		includable, err = parseIncludable(include, workingDir)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, lockfile.ProjectDir())
}
