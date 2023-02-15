// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/planner/plansdk"
	"go.jetpack.io/devbox/internal/plugin"
)

//go:embed tmpl/* tmpl/.*
var tmplFS embed.FS

var shellFiles = []string{"development.nix", "shell.nix"}

func generateForShell(rootPath string, plan *plansdk.ShellPlan, pluginManager *plugin.Manager) error {
	outPath := filepath.Join(rootPath, ".devbox/gen")

	for _, file := range shellFiles {
		err := writeFromTemplate(outPath, plan, file)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Gitignore file is added to the .devbox directory
	err := writeFromTemplate(filepath.Join(rootPath, ".devbox"), plan, ".gitignore")
	if err != nil {
		return errors.WithStack(err)
	}

	err = makeFlakeFiles(outPath, plan)
	if err != nil {
		return errors.WithStack(err)
	}

	for name, content := range plan.GeneratedFiles {
		filePath := filepath.Join(outPath, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return errors.WithStack(err)
		}
	}

	for _, pkg := range plan.DevPackages {
		if err := pluginManager.CreateFilesAndShowReadme(pkg, rootPath); err != nil {
			return err
		}
	}

	return nil
}

// writeFromTemplate writes a file in `path` directory location,
// using the template specified by `tmplName`. `tmplName` is a filepath within the
// `tmpl` directory in the devbox code.
func writeFromTemplate(path string, plan *plansdk.ShellPlan, tmplName string) error {
	embeddedPath := fmt.Sprintf("%s.tmpl", filepath.Join("tmpl", tmplName))

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
	// We use ExecuteTemplate instead of Execute because we need to identify a template that may be
	// in a sub-directory in tmplFS.
	return errors.WithStack(t.ExecuteTemplate(f, filepath.Base(tmplName+".tmpl"), plan))
}

func toJSON(a any) string {
	data, err := cuecfg.MarshalJSON(a)
	if err != nil {
		panic(err)
	}
	return string(data)
}

var templateFuncs = template.FuncMap{
	"json":                toJSON,
	"contains":            strings.Contains,
	"debug":               debug.IsEnabled,
	"isPhpRelatedPackage": isPhpRelatedPackage,
	"unifiedEnv":          featureflag.UnifiedEnv.Enabled,
}

func makeFlakeFiles(outPath string, plan *plansdk.ShellPlan) error {

	if featureflag.Flakes.Disabled() {
		return nil
	}

	flakeDir := filepath.Join(outPath, "flake")
	if err := writeFlakeFile(flakeDir, plan, "shell"); err != nil {
		return err
	}

	if hasPhpRelatedPackage(plan.DevPackages) {
		if err := writeFlakeFile(flakeDir, plan, "php"); err != nil {
			return err
		}
	} else {
		// if an old php flake file exists, then clean it up
		// deliberately ignore error since this is best effort
		_ = os.Remove(filepath.Join(flakeDir, "php", "flake.nix"))
	}
	return nil
}

// writeFlakeFile will generate a flake.nix file using the template at `tmplSubDir` in the
// devbox code.
//
// If the user's devbox project is within a git repo, then nix requires that it be tracked by git.
// We do not want the user to actually need to track the generated flake.nix in their git repo.
// So, as a workaround, we generate a temporary git repo and track it there.
func writeFlakeFile(path string, plan *plansdk.ShellPlan, tmplSubDir string) error {

	err := writeFromTemplate(path, plan, filepath.Join(tmplSubDir, "flake.nix"))
	if err != nil {
		return errors.WithStack(err)
	}

	if !isProjectInGitRepo(path) {
		// if we are not in a git repository, then carry on
		return nil
	}
	// if we are in a git repository, then nix requires that the flake.nix file be tracked by git

	// make an empty git repo
	// Alternatively consider: git add intent-to-add path/to/flake.nix, and
	// git update-index --assume-unchanged path/to/flake.nix
	// https://nixos.wiki/wiki/Flakes#How_to_add_a_file_locally_in_git_but_not_include_it_in_commits
	cmd := exec.Command("git", "-C", path, "init")
	if debug.IsEnabled() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}

	// add the flake.nix file to git
	cmd = exec.Command("git", "-C", path, "add", filepath.Join(tmplSubDir, "flake.nix"))
	if debug.IsEnabled() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	err = cmd.Run()
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func isProjectInGitRepo(dir string) bool {

	for dir != "/" {
		// Look for a .git directory in `dir`
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			// Found a .git
			return true
		} else if !os.IsNotExist(err) {
			// An error means we will not find a git repo so return false
			return false
		} else {
			// No .git directory found, so loop again into the parent dir
			dir = filepath.Dir(dir)
			continue
		}
	}
	// We reached the fs-root dir, climbed the highest mountain and
	// we still haven't found what we're looking for.
	return false
}
