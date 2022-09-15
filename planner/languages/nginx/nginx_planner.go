// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nginx

import (
	_ "embed"
	"fmt"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/planner/plansdk"
)

//go:embed shell-helper-nginx.conf
var shellHelperNginxConfig []byte

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
		ShellWelcomeMessage: fmt.Sprintf(welcomeMessage, p.shellConfig(srcDir)),
		DevPackages: []string{
			"nginx",
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
		Definitions: []string{
			fmt.Sprintf(nginxShellStartScript, srcDir, p.shellConfig(srcDir)),
		},
		GeneratedFiles: map[string][]byte{
			"shell-helper-nginx.conf": shellHelperNginxConfig,
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

var welcomeMessage = `
##### WARNING: nginx planner is experimental #####

You may need to add 

\"include ./.devbox/gen/shell-helper-nginx.conf;\" 

to your %s file to ensure the server can start in the nix shell.

Use "shell-nginx" to start the server
`
var startCommand = strings.TrimSpace(`
	addgroup --system --gid 101 nginx && \
	adduser --system --ingroup nginx --no-create-home --home /nonexistent --gecos "nginx user" --shell /bin/false --uid 101 nginx && \
	mkdir -p /var/cache/nginx/client_body && \
	mkdir -p /var/log/nginx/ && \
	echo Starting nginx with command \"nginx -c /app/%[1]s -g 'daemon off;'\" && \
	nginx -c /app/%[1]s -g 'daemon off;'
`)

const nginxShellStartScript = `
shell-nginx = pkgs.writeShellScriptBin "shell-nginx" ''

echo "Starting nginx with command:"
echo "nginx -p %[1]s -c %[2]s -e /tmp/error.log -g \"pid /tmp/mynginx.pid;daemon off;\""
nginx -p %[1]s -c %[2]s -e /tmp/error.log -g "pid /tmp/shell-nginx.pid;daemon off;"
'';`
