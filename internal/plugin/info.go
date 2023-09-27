// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plugin

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"runtime/trace"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/devpkg"
)

func Readme(ctx context.Context,
	pkg *devpkg.Package,
	projectDir string,
	markdown bool,
) (string, error) {
	defer trace.StartRegion(ctx, "Readme").End()

	cfg, err := getConfigIfAny(pkg, projectDir)
	if err != nil {
		return "", err
	}

	if cfg == nil {
		return "", nil
	}

	buf := bytes.NewBuffer(nil)

	_, _ = fmt.Fprintln(buf, "")

	if err = printReadme(cfg, buf, markdown); err != nil {
		return "", err
	}

	if err = printServices(cfg, buf, markdown); err != nil {
		return "", err
	}

	if err = printCreateFiles(cfg, buf, markdown); err != nil {
		return "", err
	}

	if err = printEnv(cfg, buf, markdown); err != nil {
		return "", err
	}

	if err = printInfoInstructions(pkg.CanonicalName(), buf); err != nil {
		return "", err
	}

	return buf.String(), nil
}

func printReadme(cfg *config, w io.Writer, markdown bool) error {
	if cfg.Readme == "" {
		return nil
	}
	_, err := fmt.Fprintf(
		w,
		"%s%s NOTES:\n%s\n\n",
		lo.Ternary(markdown, "### ", ""),
		cfg.Name,
		cfg.Readme,
	)
	return errors.WithStack(err)
}

func printServices(cfg *config, w io.Writer, markdown bool) error {
	svcs, err := cfg.Services()
	if err != nil {
		return errors.WithStack(err)
	}
	if len(svcs) == 0 {
		return nil
	}
	services := ""
	for _, service := range svcs {
		services += fmt.Sprintf("* %[1]s\n", service.Name)
	}

	_, err = fmt.Fprintf(
		w,
		"%sServices:\n%s\nUse `devbox services start|stop [service]` to interact with services\n\n",
		lo.Ternary(markdown, "### ", ""),
		services,
	)
	return errors.WithStack(err)
}

func printCreateFiles(cfg *config, w io.Writer, markdown bool) error {
	if len(cfg.CreateFiles) == 0 {
		return nil
	}

	shims := ""
	for name, src := range cfg.CreateFiles {
		if src != "" {
			shims += fmt.Sprintf("* %s\n", name)
		}
	}

	_, err := fmt.Fprintf(
		w,
		"%sThis plugin creates the following helper files:\n%s\n",
		lo.Ternary(markdown, "### ", ""),
		shims,
	)
	return errors.WithStack(err)
}

func printEnv(cfg *config, w io.Writer, markdown bool) error {
	if len(cfg.Env) == 0 {
		return nil
	}

	envVars := ""
	for name, value := range cfg.Env {
		envVars += fmt.Sprintf("* %s=%s\n", name, value)
	}

	_, err := fmt.Fprintf(
		w,
		"%sThis plugin sets the following environment variables:\n%s\n",
		lo.Ternary(markdown, "### ", ""),
		envVars,
	)
	return errors.WithStack(err)
}

func printInfoInstructions(pkg string, w io.Writer) error {
	_, err := fmt.Fprintf(
		w,
		"To show this information, run `devbox info %s`\n\n",
		pkg,
	)
	return errors.WithStack(err)
}
