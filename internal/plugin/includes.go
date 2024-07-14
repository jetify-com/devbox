package plugin

import (
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
)

func LoadConfigFromInclude(path string, plugin configfile.Plugin, lockfile *lock.File, workingDir string) (*Config, error) {
	var includable Includable
	var err error

	if plugin.Protocol == "builtin" {
		includable = devpkg.PackageFromStringWithDefaults(path, lockfile)
	} else {
		includable, err = parseIncludable(path, plugin, workingDir)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, lockfile.ProjectDir())
}
