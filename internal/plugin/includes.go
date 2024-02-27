package plugin

import (
	"strings"

	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
)

func LoadConfigFromInclude(include, projectDir string) (*Config, error) {
	var includable Includable
	var err error
	if t, name, _ := strings.Cut(include, ":"); t == "plugin" {
		includable = devpkg.PackageFromStringWithDefaults(
			name,
			&lock.DummyLocker{ProjectDirVal: projectDir},
		)
	} else {
		includable, err = parseReflike(include, projectDir)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, projectDir)
}
