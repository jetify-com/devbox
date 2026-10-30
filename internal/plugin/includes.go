package plugin

import (
	"strings"

	"go.jetify.com/devbox/internal/boxcli/usererr"
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
	config, err := getConfigIfAny(includable, lockfile.ProjectDir())
	if err == nil && config == nil {
		return nil, usererr.New("No built-in plugin found for include %q", include)
	}
	return config, err
}
