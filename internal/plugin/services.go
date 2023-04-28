// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"go.jetpack.io/devbox/internal/services"
)

func GetServices(pkgs []string, projectDir string) (services.Services, error) {
	svcs := services.Services{}
	for _, pkg := range pkgs {
		conf, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if conf == nil {
			continue
		}

		if file, hasProcessComposeYaml := conf.ProcessComposeYaml(); hasProcessComposeYaml {
			svc := services.Service{
				Name:               conf.Name,
				Env:                conf.Env,
				ProcessComposePath: file,
			}
			svcs[conf.Name] = svc
		}

	}
	return svcs, nil
}
