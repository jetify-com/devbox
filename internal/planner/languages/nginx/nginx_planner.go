// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"go.jetpack.io/devbox/internal/planner/plansdk"
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

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	return &plansdk.ShellPlan{
		DevPackages: []string{
			"nginx",
			"shell-nginx",
		},
		Definitions: []string{
			fmt.Sprintf(nginxShellStartScript, srcDir, p.shellConfig(srcDir)),
		},
		GeneratedFiles: map[string]string{
			"shell-helper-nginx.conf": fmt.Sprintf(shellHelperNginxConfig, os.TempDir()),
		},
		ShellInitHook: []string{
			plansdk.WelcomeMessage(fmt.Sprintf(welcomeMessage, p.shellConfig(srcDir)))},
	}
}

func (p *Planner) shellConfig(srcDir string) string {
	if plansdk.FileExists(filepath.Join(srcDir, "shell-nginx.conf")) {
		return "shell-nginx.conf"
	}
	return "nginx.conf"
}

const welcomeMessage = `
##### WARNING: nginx planner is experimental #####

You may need to add

\"include ./.devbox/gen/shell-helper-nginx.conf;\"

to your %s file to ensure the server can start in the nix shell.

Use \"shell-nginx\" to start the server
`

const nginxShellStartScript = `
shell-nginx = pkgs.writeShellScriptBin "shell-nginx" ''

echo "Starting nginx with command:"
echo "nginx -p %[1]s -c %[2]s -e /tmp/error.log -g \"pid /tmp/mynginx.pid;daemon off;\""
nginx -p %[1]s -c %[2]s -e /tmp/error.log -g "pid /tmp/shell-nginx.pid;daemon off;"
'';`

const shellHelperNginxConfig = `access_log %[1]s/access.log;
client_body_temp_path %[1]s/client_body;
proxy_temp_path %[1]s/proxy;
fastcgi_temp_path %[1]s/fastcgi;
uwsgi_temp_path %[1]s/uwsgi;
scgi_temp_path %[1]s/scgi;
`
