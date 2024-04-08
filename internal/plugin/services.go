// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"fmt"
	"os"

	"go.jetpack.io/devbox/internal/services"
)

func GetServices(configs []*Config) (services.Services, error) {
	allSvcs := services.Services{}
	for _, conf := range configs {
		svcs, err := conf.Services()
		if err != nil {
			fmt.Fprintf(
				os.Stderr,
				"error reading services in plugin \"%s\", skipping",
				conf.Name,
			)
			continue
		}
		for name, svc := range svcs {
			allSvcs[name] = svc
		}
	}

	return allSvcs, nil
}
