// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	"fmt"
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
	fmt.Println(srcDir)
	return &plansdk.Plan{
		DevPackages: []string{
			"custom-nginx",
			"shell-nginx",
		},
		RuntimePackages: []string{
			"nginx",
		},
		InstallStage: &plansdk.Stage{
			InputFiles: plansdk.AllFiles(),
		},
		StartStage: &plansdk.Stage{
			// These 2 directories are required and are not created by nginx?
			Command: "mkdir -p /var/cache/nginx/client_body && " +
				"mkdir -p /var/log/nginx/ && " +
				"nginx -c /app/nginx.conf -g 'daemon off;'",
			InputFiles: plansdk.AllFiles(),
		},
		// These definitions are only used in dev.
		Definitions: []string{
			customNginxDefintion,
			fmt.Sprintf(nginxShellStartScript, srcDir),
			// nginxOverwriteDefintion,
		},
	}
}

const customNginxDefintion = `
custom-nginx = pkgs.nginx.overrideAttrs (oldAttrs: rec {
	configureFlags = oldAttrs.configureFlags ++ [
			"--http-client-body-temp-path=/tmp/cache/client_body"
			"--http-proxy-temp-path=/tmp/cache/proxy"
			"--http-fastcgi-temp-path=/tmp/cache/fastcgi"
			"--http-uwsgi-temp-path=/tmp/cache/uwsgi"
			"--http-scgi-temp-path=/tmp/cache/scgi"
		];
});`

const nginxShellStartScript = `
shell-nginx = pkgs.writeShellScriptBin "shell-nginx" ''
	echo "Starting nginx with command:"
	echo "nginx -p %[1]s -c shell-nginx.conf -e /tmp/error.log -g \"pid /tmp/mynginx.pid;daemon off;\""
  nginx -p %[1]s -c shell-nginx.conf -e /tmp/error.log -g "pid /tmp/mynginx.pid;daemon off;"
'';`

const nginxOverwriteDefintion = `
nginx = pkgs.writeScriptBin "nginx" ''
	exec ${nginxCustom}/bin/nginx -c ${./../../shell-nginx.conf} -e /tmp/error.log -g "pid /tmp/mynginx.pid;daemon off;"
'';`
