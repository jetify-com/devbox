// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"os"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg Includable, projectDir string) (*config, error) {
	switch pkg := pkg.(type) {
	case *devpkg.Package:
		return getBuiltinPluginConfig(pkg, projectDir)
	case *githubPlugin:
		return getConfigFromGithub(pkg, projectDir)
	case *localPlugin:
		content, err := os.ReadFile(pkg.path)
		if err != nil && !os.IsNotExist(err) {
			return nil, errors.WithStack(err)
		}
		return buildConfig(pkg, projectDir, string(content))
	}
	return nil, errors.Errorf("unknown plugin type %T", pkg)
}

func getBuiltinPluginConfig(pkg Includable, projectDir string) (*config, error) {
	builtins, err := plugins.Builtins()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, file := range builtins {
		// We deserialize first so we can check the Match field. If it's there
		// we use it, otherwise we use the name.
		// TODO(landau): this is weird, hard to understand code. Fetching the file
		// content of configs should probably not use FileContent()
		content, err := pkg.FileContent(file.Name())
		if err != nil {
			return nil, errors.WithStack(err)
		}

		name := pkg.CanonicalName()
		cfg, err := buildConfig(pkg, projectDir, string(content))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// if match regex is set we use it to check. Otherwise we assume it's a
		// perfect match
		if (cfg.Match != "" && !regexp.MustCompile(cfg.Match).MatchString(name)) ||
			(cfg.Match == "" && strings.Split(file.Name(), ".")[0] != name) {
			continue
		}
		return cfg, nil
	}
	return nil, nil
}

func getConfigFromGithub(pkg *githubPlugin, projectDir string) (*config, error) {
	content, err := pkg.FileContent("devbox.json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(pkg, projectDir, string(content))
}
