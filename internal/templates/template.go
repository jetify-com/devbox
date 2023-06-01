// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package templates

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"golang.org/x/exp/slices"
)

func Init(w io.Writer, template, dir string) error {
	if err := createDirAndEnsureEmpty(dir); err != nil {
		return err
	}

	templatePath, ok := templates[template]
	if !ok {
		return usererr.New("unknown template %q", template)
	}

	tmp, err := os.MkdirTemp("", "devbox-template")
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(
		"git", "clone", "https://github.com/jetpack-io/devbox.git", tmp,
	)
	fmt.Fprintf(w, "%s\n", cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	cmd = exec.Command(
		"sh", "-c",
		fmt.Sprintf("cp -r %s %s", filepath.Join(tmp, templatePath, "*"), dir),
	)
	fmt.Fprintf(w, "%s\n", cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return errors.WithStack(cmd.Run())
}

func List(w io.Writer, showAll bool) {
	fmt.Fprintf(w, "Templates:\n\n")
	keysToShow := popularTemplates
	if showAll {
		keysToShow = lo.Keys(templates)
	}

	slices.Sort(keysToShow)
	for _, key := range keysToShow {
		fmt.Fprintf(w, "* %-15s %s\n", key, templates[key])
	}
}

func createDirAndEnsureEmpty(dir string) error {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		if err = os.MkdirAll(dir, 0755); err != nil {
			return errors.WithStack(err)
		}
	} else if err != nil {
		return errors.WithStack(err)
	}

	if len(entries) > 0 {
		return usererr.New("directory %q is not empty", dir)
	}

	return nil
}
