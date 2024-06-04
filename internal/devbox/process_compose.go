// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/devbox/devopt"
	"go.jetpack.io/devbox/internal/xdg"
)

const processComposeVersion = "1.5.0"

var processComposeConfigPath string

func InitializeProcessCompose(ctx context.Context, stderr io.Writer) (string, error) {
	pcDevboxProjectPath, err := ensurePcDevboxProject()
	if err != nil {
		return "", err
	}

	box, err := Open(&devopt.Opts{
		Dir:    pcDevboxProjectPath,
		Stderr: stderr,
	})
	if err != nil {
		return "", errors.WithStack(err)
	}

	if err = box.Add(ctx, []string{"process-compose@" + processComposeVersion}, devopt.AddOpts{}); err != nil {
		return "", err
	}

	err = box.Install(ctx)
	if err != nil {
		return "", err
	}

	return utilityLookPath("process-compose")
}

func ensurePcDevboxProject() (string, error) {
	if processComposeConfigPath != "" {
		return processComposeConfigPath, nil
	}

	pcDevboxProjectDir, err := createPcDevboxProjectDir()
	if err != nil {
		return "", err
	}

	_, err = InitConfig(pcDevboxProjectDir)
	if err != nil {
		return "", err
	}

	return pcDevboxProjectDir, nil
}

func createPcDevboxProjectDir() (string, error) {
	path := xdg.DataSubpath(filepath.Join("devbox/util/process-compose"))
	if err := os.MkdirAll(path, 0o755); err != nil {
		return "", errors.WithStack(err)
	}
	return path, nil
}
