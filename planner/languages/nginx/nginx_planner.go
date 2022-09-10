// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	"path/filepath"

	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "nginx.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "nginx.conf"))
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	return &plansdk.Plan{
		DevPackages: []string{
			"nginx",
		},
		RuntimePackages: []string{
			"nginx",
		},
		SharedPlan: plansdk.SharedPlan{
			StartStage: &plansdk.Stage{
				// These 2 directories are required and are not created by nginx?
				Command: "mkdir -p /var/cache/nginx/client_body && " +
					"mkdir -p /var/log/nginx/ && " +
					"nginx -c /app/nginx.conf -g 'daemon off;'",
			},
		},
	}
}
