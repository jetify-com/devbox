// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/planner"
)

//go:embed tmpl/* tmpl/.*
var tmplFS embed.FS

func generate(rootPath string, plan *planner.BuildPlan) error {
	// TODO: we should also generate a .dockerignore file
	files := []string{".gitignore", "Dockerfile", "shell.nix", "default.nix"}

	outPath := filepath.Join(rootPath, ".devbox/gen")

	for _, file := range files {
		err := writeFromTemplate(outPath, plan, file)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	return nil
}

func writeFromTemplate(path string, plan *planner.BuildPlan, tmplName string) error {
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
	return t.Execute(f, plan)
}

func toJSON(a any) string {
	data, _ := json.Marshal(a)
	return string(data)
}

var templateFuncs = template.FuncMap{
	"json": toJSON,
}
