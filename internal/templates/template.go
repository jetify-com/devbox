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

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/build"
)

const devboxRepo = "https://github.com/jetify-com/devbox"

func InitFromName(w io.Writer, template, target string) error {
	templatePath, ok := templates[template]
	if !ok {
		return usererr.New("unknown template name or format %q", template)
	}
	return InitFromRepo(w, devboxRepo, templatePath, target)
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
	defer os.RemoveAll(tmp)

	args := []string{"clone", parsedRepoURL}
	// Built-in templates are checked out at the ref matching this devbox version.
	if parsedRepoURL == devboxRepo {
		args = append(args, "-b", lo.Ternary(build.IsDev, "main", build.Version))
	}
	// Create shallow clone with depth of 1
	args = append(args, "--depth", "1", tmp)
	cmd := exec.Command("git", args...)
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
	// like: https://github.com/jetify-com/devbox.git
	return strings.TrimSuffix(repo, ".git"), nil
}
