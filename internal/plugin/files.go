// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg *nix.Package, projectDir string) (*config, error) {
	configFiles, err := plugins.BuiltIn.ReadDir(".")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, file := range configFiles {
		if file.IsDir() || strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		// Try to find perfect match first
		content, err := plugins.BuiltIn.ReadFile(file.Name())
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

func getFileContent(contentPath string) ([]byte, error) {
	return plugins.BuiltIn.ReadFile(contentPath)
}
