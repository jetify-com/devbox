package plugin

import (
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/nix/flake"
)

func LoadConfigFromInclude(path string, ref flake.Ref, lockfile *lock.File, workingDir string) (*Config, error) {
	var includable Includable
	var err error

	if ref.Type == "builtin" {
		includable = devpkg.PackageFromStringWithDefaults(path, lockfile)
	} else {
		includable, err = parseIncludable(path, ref, workingDir)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, lockfile.ProjectDir())
}
