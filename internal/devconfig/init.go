// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/devpkg/pkgtype"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec"
)

func Init(dir string, writer io.Writer) (created bool, err error) {
	created, err = initConfigFile(filepath.Join(dir, configfile.DefaultName))
	if err != nil || !created {
		return created, err
	}

	// package suggestion
	pkgsToSuggest, err := initrec.Get(dir)
	if err != nil {
		return created, err
	}
	if len(pkgsToSuggest) > 0 {
		s := fmt.Sprintf("devbox add %s", strings.Join(pkgsToSuggest, " "))
		fmt.Fprintf(
			writer,
			"We detected extra packages you may need. To install them, run `%s`\n",
			color.HiYellowString(s),
		)
	}
	return created, err
}

func initConfigFile(path string) (created bool, err error) {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o644)
	if errors.Is(err, os.ErrExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer func() {
		if err != nil {
			os.Remove(file.Name())
		}
	}()

	_, err = file.Write(DefaultConfig().Root.Bytes())
	if err != nil {
		file.Close()
		return false, err
	}
	if err := file.Close(); err != nil {
		return false, err
	}
	return true, nil
}

func Open(projectDir string) (*Config, error) {
	cfgPath := filepath.Join(projectDir, configfile.DefaultName)

	if !featureflag.TySON.Enabled() {
		return readFromFile(cfgPath)
	}

	tysonCfgPath := filepath.Join(projectDir, configfile.DefaultTySONName)

	// If tyson config exists use it. Otherwise fallback to json config.
	// In the future we may error out if both configs exist, but for now support
	// both while we experiment with tyson support.
	if fileutil.Exists(tysonCfgPath) {
		paths, err := pkgtype.RunXClient().Install(context.TODO(), "jetpack-io/tyson")
		if err != nil {
			return nil, err
		}
		binPath := filepath.Join(paths[0], "tyson")
		tmpFile, err := os.CreateTemp("", "*.json")
		if err != nil {
			return nil, err
		}
		cmd := exec.Command(binPath, "eval", tysonCfgPath)
		cmd.Stderr = os.Stderr
		cmd.Stdout = tmpFile
		if err = cmd.Run(); err != nil {
			return nil, err
		}
		cfgPath = tmpFile.Name()
		config, err := readFromFile(cfgPath)
		if err != nil {
			return nil, err
		}
		config.Root.Format = configfile.TSONFormat
		return config, nil
	}

	return readFromFile(cfgPath)
}
