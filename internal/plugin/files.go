// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"io/fs"
	"os"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg Includable, projectDir string) (*config, error) {
	switch pkg := pkg.(type) {
	case *devpkg.Package:
		return getBuiltinPluginConfigIfExists(pkg, projectDir)
	case *githubPlugin:
		return pkg.buildConfig(projectDir)
	case *localPlugin:
		content, err := os.ReadFile(pkg.path)
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
) (*config, error) {
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
