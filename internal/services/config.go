// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package services

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/f1bonacc1/process-compose/src/types"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/cuecfg"
)

func FromUserProcessCompose(projectDir, userProcessCompose string) Services {
	processComposeYaml := lookupProcessCompose(projectDir, userProcessCompose)
	if processComposeYaml == "" {
		return nil
	}

	userSvcs, err := FromProcessCompose(processComposeYaml)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error reading process-compose.yaml: %s, skipping", err)
		return nil
	}
	return userSvcs
}

func FromProcessCompose(path string) (Services, error) {
	processCompose := &types.Project{}
	services := Services{}
	err := errors.WithStack(cuecfg.ParseFile(path, processCompose))
	if err != nil {
		return nil, err
	}

	for name := range processCompose.Processes {
		svc := Service{
			Name:               name,
			ProcessComposePath: path,
		}
		services[name] = svc
	}

	return services, nil
}

func lookupProcessCompose(projectDir, path string) string {
	if path == "" {
		path = projectDir
	}
	if !filepath.IsAbs(path) {
		path = filepath.Join(projectDir, path)
	}

	pathsToCheck := []string{
		path,
		filepath.Join(path, "process-compose.yaml"),
		filepath.Join(path, "process-compose.yml"),
	}

	for _, p := range pathsToCheck {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p
		}
	}

	return ""
}
