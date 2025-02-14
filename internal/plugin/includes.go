package plugin

import (
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/nix/flake"
)

func LoadConfigFromInclude(ref flake.Ref, lockfile *lock.File, workingDir string) (*Config, error) {
	var includable Includable
	var err error

	if ref.Type == flake.TypeBuiltin {
		includable = devpkg.PackageFromStringWithDefaults(ref.Path, lockfile)
	} else {
		includable, err = parseIncludable(ref, workingDir)
		if err != nil {
			return nil, err
		}
	}
	return getConfigIfAny(includable, lockfile.ProjectDir())
}
