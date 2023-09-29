// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellgen

import (
	"bytes"
	"context"
	"embed"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/trace"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/debug"
)

//go:embed tmpl/*
var tmplFS embed.FS

// GenerateForPrintEnv will create all the files necessary for processing
// devbox.PrintEnv, which is the core function from which devbox shell/run/direnv
// functionality is derived.
func GenerateForPrintEnv(ctx context.Context, devbox devboxer) error {
	defer trace.StartRegion(ctx, "generateShellFiles").End()

	plan, err := newFlakePlan(ctx, devbox)
	if err != nil {
		return err
	}

	outPath := genPath(devbox)

	// Preserving shell.nix to avoid breaking old-style .envrc users
	err = writeFromTemplate(outPath, plan, "shell.nix", "shell.nix")
	if err != nil {
		return errors.WithStack(err)
	}

	// Gitignore file is added to the .devbox directory
	err = writeFromTemplate(filepath.Join(devbox.ProjectDir(), ".devbox"), plan, ".gitignore", ".gitignore")
	if err != nil {
		return errors.WithStack(err)
	}

	err = makeFlakeFile(devbox, outPath, plan)
	if err != nil {
		return errors.WithStack(err)
	}

	return WriteScriptsToFiles(devbox)
}

// Cache and buffers for generating templated files.
var (
	tmplCache = map[string]*template.Template{}

	// Most generated files are < 4KiB.
	tmplNewBuf = bytes.NewBuffer(make([]byte, 0, 4096))
	tmplOldBuf = bytes.NewBuffer(make([]byte, 0, 4096))
)

func writeFromTemplate(path string, plan any, tmplName, generatedName string) error {
	tmplKey := tmplName + ".tmpl"
	tmpl := tmplCache[tmplKey]
	if tmpl == nil {
		tmpl = template.New(tmplKey)
		tmpl.Funcs(templateFuncs)

		var err error
		tmpl, err = tmpl.ParseFS(tmplFS, "tmpl/"+tmplKey)
		if err != nil {
			return errors.WithStack(err)
		}
		tmplCache[tmplKey] = tmpl
	}
	tmplNewBuf.Reset()
	if err := tmpl.Execute(tmplNewBuf, plan); err != nil {
		return errors.WithStack(err)
	}

	// In some circumstances, Nix looks at the mod time of a file when
	// caching, so we only want to update the file if something has
	// changed. Blindly overwriting the file could invalidate Nix's cache
	// every time, slowing down evaluation considerably.
	var (
		outPath = filepath.Join(path, generatedName)
		flag    = os.O_RDWR | os.O_CREATE
		perm    = fs.FileMode(0o644)
	)
	outFile, err := os.OpenFile(outPath, flag, perm)
	if errors.Is(err, fs.ErrNotExist) {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return errors.WithStack(err)
		}
		outFile, err = os.OpenFile(outPath, flag, perm)
	}
	if err != nil {
		return errors.WithStack(err)
	}
	defer outFile.Close()

	// Only read len(tmplWriteBuf) + 1 from the existing file so we can
	// check if the lengths are different without reading the whole thing.
	tmplOldBuf.Reset()
	tmplOldBuf.Grow(tmplNewBuf.Len() + 1)
	_, err = io.Copy(tmplOldBuf, io.LimitReader(outFile, int64(tmplNewBuf.Len())+1))
	if err != nil {
		return errors.WithStack(err)
	}
	if bytes.Equal(tmplNewBuf.Bytes(), tmplOldBuf.Bytes()) {
		return nil
	}

	// Replace the existing file contents.
	if _, err := outFile.Seek(0, io.SeekStart); err != nil {
		return errors.WithStack(err)
	}
	if err := outFile.Truncate(int64(tmplNewBuf.Len())); err != nil {
		return errors.WithStack(err)
	}
	if _, err := io.Copy(outFile, tmplNewBuf); err != nil {
		return errors.WithStack(err)
	}
	return errors.WithStack(outFile.Close())
}

func toJSON(a any) string {
	data, err := cuecfg.MarshalJSON(a)
	if err != nil {
		panic(err)
	}
	return string(data)
}

var templateFuncs = template.FuncMap{
	"json":     toJSON,
	"contains": strings.Contains,
	"debug":    debug.IsEnabled,
}

func makeFlakeFile(d devboxer, outPath string, plan *flakePlan) error {
	flakeDir := FlakePath(d)
	templateName := "flake.nix"
	if featureflag.RemoveNixpkgs.Enabled() {
		templateName = "flake_remove_nixpkgs.nix"
	}
	err := writeFromTemplate(flakeDir, plan, templateName, "flake.nix")
	if err != nil {
		return errors.WithStack(err)
	}

	if !isProjectInGitRepo(outPath) {
		// if we are not in a git repository, then carry on
		return nil
	}
	// if we are in a git repository, then nix requires that the flake.nix file be tracked by git

	// make an empty git repo
	// Alternatively consider: git add intent-to-add path/to/flake.nix, and
	// git update-index --assume-unchanged path/to/flake.nix
	// https://nixos.wiki/wiki/Flakes#How_to_add_a_file_locally_in_git_but_not_include_it_in_commits
	cmd := exec.Command("git", "-C", flakeDir, "init")
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
	cmd = exec.Command("git", "-C", flakeDir, "add", "flake.nix")
	if debug.IsEnabled() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}
	return errors.WithStack(cmd.Run())
}

func isProjectInGitRepo(dir string) bool {
	for dir != "/" {
		// Look for a .git directory in `dir`
		_, err := os.Stat(filepath.Join(dir, ".git"))
		if err == nil {
			// Found a .git
			return true
		}
		if !errors.Is(err, fs.ErrNotExist) {
			// An error means we will not find a git repo so return false
			return false
		}
		// No .git directory found, so loop again into the parent dir
		dir = filepath.Dir(dir)
	}
	// We reached the fs-root dir, climbed the highest mountain and
	// we still haven't found what we're looking for.
	return false
}
