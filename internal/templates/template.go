// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package templates

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

func InitFromName(w io.Writer, template string, target string) error {
	templatePath, ok := templates[template]
	if !ok {
		return usererr.New("unknown template name or format %q", template)
	}
	return InitFromRepo(w, "https://github.com/jetpack-io/devbox", templatePath, target)
}

func InitFromRepo(w io.Writer, repo string, subdir string, target string) error {
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
	cmd := exec.Command("git", "clone", parsedRepoURL, tmp)
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

func ParseRepoURL(repo string) (string, error) {
	u, err := url.Parse(repo)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return "", usererr.New("Invalid URL format for --repo %s", repo)
	}
	// this is to handle cases where user puts repo url with .git at the end
	// like: https://github.com/jetpack-io/devbox.git
	return strings.TrimSuffix(repo, ".git"), nil
}
