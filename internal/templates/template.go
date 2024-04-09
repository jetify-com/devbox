// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package templates

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/build"
)

func InitFromName(w io.Writer, template, target string) error {
	templatePath, ok := templates[template]
	if !ok {
		return usererr.New("unknown template name or format %q", template)
	}
	return InitFromRepo(w, "https://github.com/jetpack-io/devbox", templatePath, target)
}

func InitFromRepo(w io.Writer, repo, subdir, target string) error {
	if err := createDirAndEnsureEmpty(target); err != nil {
		return err
	}
	parsedRepoURL, err := ParseRepoURL(repo)
	if err != nil {
		return errors.WithStack(err)
	}

	tmp, err := os.MkdirTemp("", "devbox-template")
	if err != nil {
		return errors.WithStack(err)
	}
	cmd := exec.Command(
		"git", "clone", parsedRepoURL,
		// Clone and checkout a specific ref
		"-b", lo.Ternary(build.IsDev, "main", build.Version),
		tmp,
	)
	fmt.Fprintf(w, "%s\n", cmd)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	if err = cmd.Run(); err != nil {
		return errors.WithStack(err)
	}

	cmd = exec.Command(
		"sh", "-c",
		fmt.Sprintf("cp -r %s %s", filepath.Join(tmp, subdir, "*"), target),
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
		if err = os.MkdirAll(dir, 0o755); err != nil {
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

func ParseRepoURL(repo string) (string, error) {
	u, err := url.Parse(repo)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", usererr.New("Invalid URL format for --repo %s", repo)
	}
	// this is to handle cases where user puts repo url with .git at the end
	// like: https://github.com/jetpack-io/devbox.git
	return strings.TrimSuffix(repo, ".git"), nil
}
