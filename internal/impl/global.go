// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package impl

import (
	"context"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/devconfig"
	"go.jetpack.io/devbox/internal/pullbox"
	"go.jetpack.io/devbox/internal/xdg"
)

// In the future we will support multiple global profiles
const currentGlobalProfile = "default"

func (d *Devbox) PullGlobal(
	ctx context.Context,
	force bool,
	path string,
) error {
	u, err := url.Parse(path)
	if err == nil && u.Scheme != "" {
		return d.pullGlobalFromURL(ctx, force, u)
	}
	return d.pullGlobalFromPath(ctx, path)
}

func (d *Devbox) PrintGlobalList() error {
	for _, p := range d.cfg.Packages {
		fmt.Fprintf(d.writer, "* %s\n", p)
	}
	return nil
}

func (d *Devbox) pullGlobalFromURL(
	ctx context.Context,
	overwrite bool,
	configURL *url.URL,
) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", configURL)
	puller := pullbox.New()
	if ok, err := puller.URLIsArchive(configURL.String()); ok {
		fmt.Fprintf(
			d.writer,
			"%s is an archive, extracting to %s\n",
			configURL,
			d.ProjectDir(),
		)
		return puller.DownloadAndExtract(
			overwrite,
			configURL.String(),
			d.projectDir,
		)
	} else if err != nil {
		return err
	}
	cfg, err := devconfig.LoadConfigFromURL(configURL)
	if err != nil {
		return err
	}
	return d.addFromPull(ctx, cfg)
}

func (d *Devbox) pullGlobalFromPath(ctx context.Context, path string) error {
	fmt.Fprintf(d.writer, "Pulling global config from %s\n", path)
	cfg, err := devconfig.Load(path)
	if err != nil {
		return err
	}
	return d.addFromPull(ctx, cfg)
}

func (d *Devbox) addFromPull(ctx context.Context, cfg *devconfig.Config) error {
	diff, _ := lo.Difference(cfg.Packages, d.cfg.Packages)
	if len(diff) == 0 {
		fmt.Fprint(d.writer, "No new packages to install\n")
		return nil
	}
	fmt.Fprintf(
		d.writer,
		"Installing the following packages: %s\n",
		strings.Join(diff, ", "),
	)
	return d.Add(ctx, diff...)
}

func GlobalDataPath() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/global", currentGlobalProfile))
	if err := os.MkdirAll(path, 0755); err != nil {
		return "", errors.WithStack(err)
	}

	nixProfilePath := filepath.Join(path)
	currentPath := xdg.DataSubpath("devbox/global/current")

	// For now default is always current. In the future we will support multiple
	// and allow user to switch. Remove any existing symlink and create a new one
	// because previous versions of devbox may have created a symlink to a
	// different profile.
	existing, _ := os.Readlink(currentPath)
	if existing != nixProfilePath {
		_ = os.Remove(currentPath)
	}

	err := os.Symlink(nixProfilePath, currentPath)
	if err != nil && !errors.Is(err, fs.ErrExist) {
		return "", errors.WithStack(err)
	}

	return path, nil
}
