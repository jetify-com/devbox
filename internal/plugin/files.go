// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
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

func getConfigIfAny(inc Includable, projectDir string) (*Config, error) {
	switch includable := inc.(type) {
	case *devpkg.Package:
		return getBuiltinPluginConfigIfExists(includable, projectDir)
	case *githubPlugin:
		content, err := includable.Fetch()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return buildConfig(includable, projectDir, string(content))
	case *gitlabPlugin:
		content, err := includable.Fetch()
		if err != nil {
			return nil, errors.WithStack(err)
		}
		return buildConfig(includable, projectDir, string(content))
	case *LocalPlugin:
		content, err := os.ReadFile(includable.Path())
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.WithStack(err)
		}
		return buildConfig(includable, projectDir, string(content))
	}
	return nil, errors.Errorf("unknown plugin type %T", inc)
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
