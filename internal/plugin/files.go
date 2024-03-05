// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"io/fs"
	"os"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg Includable, projectDir string) (*Config, error) {
	switch pkg := pkg.(type) {
	case *devpkg.Package:
		return getBuiltinPluginConfigIfExists(pkg, projectDir)
	case *githubPlugin:
		content, err := pkg.Fetch()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return buildConfig(pkg, projectDir, string(content))
	case *localPlugin:
		content, err := os.ReadFile(pkg.Path())
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.WithStack(err)
		}
		return buildConfig(pkg, projectDir, string(content))
	}
	return nil, errors.Errorf("unknown plugin type %T", pkg)
}

func getBuiltinPluginConfigIfExists(
	pkg *devpkg.Package,
	projectDir string,
) (*Config, error) {
	if pkg.DisablePlugin {
		return nil, nil
	}
	content, err := plugins.BuiltInForPackage(pkg.CanonicalName())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(pkg, projectDir, string(content))
}

func GetBuiltinsForPackages(
	packages []configfile.Package,
	lockfile *lock.File,
) ([]*Config, error) {
	builtIns := []*Config{}
	for _, pkg := range devpkg.PackagesFromConfig(packages, lockfile) {
		config, err := getBuiltinPluginConfigIfExists(pkg, lockfile.ProjectDir())
		if err != nil {
			return nil, err
		}
		if config != nil {
			builtIns = append(builtIns, config)
		}
	}
	return builtIns, nil
}
