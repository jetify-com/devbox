// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/initrec"
)

func Init(dir string, writer io.Writer) (created bool, err error) {
	cfgPath := filepath.Join(dir, DefaultName)

	config := DefaultConfig()

	// package suggestion
	pkgsToSuggest, err := initrec.Get(dir)
	if err != nil {
		return false, err
	}
	if len(pkgsToSuggest) > 0 {
		s := fmt.Sprintf("devbox add %s", strings.Join(pkgsToSuggest, " "))
		fmt.Fprintf(
			writer,
			"We detected extra packages you may need. To install them, run `%s`\n",
			color.HiYellowString(s),
		)
	}

	return cuecfg.InitFile(cfgPath, config)
}

func Open(projectDir string) (*Config, error) {
	cfgPath := filepath.Join(projectDir, DefaultName)

	if _, err := os.Stat(filepath.Join(projectDir, DefaultTySONName)); err == nil {
		paths, err := pkgtype.RunXClient().Install(context.TODO(), "jetpack-io/tyson")
		if err != nil {
			return nil, err
		}
		binPath := filepath.Join(paths[0], "tyson")
		tmpFile, err := os.CreateTemp("", "*.json")
		if err != nil {
			return nil, err
		}
		cmd := exec.Command(binPath, "eval", filepath.Join(projectDir, DefaultTySONName))
		cmd.Stderr = os.Stderr
		cmd.Stdout = tmpFile
		if err = cmd.Run(); err != nil {
			return nil, err
		}
		cfgPath = tmpFile.Name()
	}

	return Load(cfgPath)
}
