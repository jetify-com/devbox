// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "nginx.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	return plansdk.FileExists(filepath.Join(srcDir, "nginx.conf")) ||
		plansdk.FileExists(filepath.Join(srcDir, "shell-nginx.conf"))
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	fmt.Println(srcDir)
	return &plansdk.Plan{
		ShellWelcomeMessage: "\n##### WARNING: nginx planner is experimental #####\n\nUse \"shell-nginx\" to start the server\n",
		ShellPackages: []string{
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
			// Create user/group and directories
			Command:    fmt.Sprintf(startCommand, p.buildConfig(srcDir)),
			InputFiles: plansdk.AllFiles(),
		},
		// These definitions are only used in shell. Since nix is lazy, it won't
		// actually build them at all if they are not used.
		Definitions: []string{
			customNginxDefintion,
			fmt.Sprintf(nginxShellStartScript, srcDir, p.shellConfig(srcDir)),
		},
	}
}

func (p *Planner) shellConfig(srcDir string) string {
	if plansdk.FileExists(filepath.Join(srcDir, "shell-nginx.conf")) {
		return "shell-nginx.conf"
	}
	return "nginx.conf"
}

func (p *Planner) buildConfig(srcDir string) string {
	if plansdk.FileExists(filepath.Join(srcDir, "nginx.conf")) {
		return "nginx.conf"
	}
	return "shell-nginx.conf"
}

var startCommand = strings.TrimSpace(`
	addgroup --system --gid 101 nginx && \
	adduser --system --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx && \
	mkdir -p /var/cache/nginx/client_body && \
	mkdir -p /var/log/nginx/ && \
	echo Starting nginx with command \"nginx -c /app/%[1]s -g 'daemon off;'\" && \
	nginx -c /app/%[1]s -g 'daemon off;'
`)

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
echo "nginx -p %[1]s -c %[2]s -e /tmp/error.log -g \"pid /tmp/mynginx.pid;daemon off;\""
nginx -p %[1]s -c %[2]s -e /tmp/error.log -g "pid /tmp/shell-nginx.pid;daemon off;"
'';`
