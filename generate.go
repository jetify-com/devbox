// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/boxcli/featureflag"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/pkgcfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

//go:embed tmpl/* tmpl/.*
var tmplFS embed.FS

var shellFiles = []string{"development.nix", "shell.nix"}
var buildFiles = []string{"development.nix", "runtime.nix", "Dockerfile", "Dockerfile.dockerignore"}

func generateForShell(rootPath string, plan *plansdk.ShellPlan) error {
	outPath := filepath.Join(rootPath, ".devbox/gen")

	for _, file := range shellFiles {
		err := writeFromTemplate(outPath, plan, file)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Gitignore file is added to the .devbox directory
	// TODO savil. Remove this hardcode from here, so this function can be generically defined again
	//    by accepting the files list parameter.
	err := writeFromTemplate(filepath.Join(rootPath, ".devbox"), plan, ".gitignore")
	if err != nil {
		return errors.WithStack(err)
	}

	for name, content := range plan.GeneratedFiles {
		filePath := filepath.Join(outPath, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return errors.WithStack(err)
		}
	}

	if featureflag.PKGConfig.Enabled() {
		for _, pkg := range plan.DevPackages {
			if err := pkgcfg.CreateFilesAndShowReadme(pkg, rootPath); err != nil {
				return err
			}
		}
	}

	return nil
}

func generateForBuild(rootPath string, plan *plansdk.BuildPlan) error {
	outPath := filepath.Join(rootPath, ".devbox/gen")

	for _, file := range buildFiles {
		err := writeFromTemplate(outPath, plan, file)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func writeFromTemplate(path string, plan interface{}, tmplName string) error {
	embeddedPath := fmt.Sprintf("tmpl/%s.tmpl", tmplName)

	// Should we clear the directory so we start "fresh"?
	outPath := filepath.Join(path, tmplName)
	outDir := filepath.Dir(outPath)
	err := os.MkdirAll(outDir, 0755) // Ensure directory exists.
	if err != nil {
		return errors.WithStack(err)
	}

	f, err := os.Create(outPath)
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		return errors.WithStack(err)
	}
	t := template.Must(template.New(tmplName+".tmpl").Funcs(templateFuncs).ParseFS(tmplFS, embeddedPath))
	return errors.WithStack(t.Execute(f, plan))
}

func toJSON(a any) string {
	data, _ := json.Marshal(a)
	return string(data)
}

var templateFuncs = template.FuncMap{
	"json":     toJSON,
	"contains": strings.Contains,
	"debug":    debug.IsEnabled,
}
