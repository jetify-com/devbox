// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devconfig

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/fatih/color"

	"go.jetpack.io/devbox/internal/initrec"
)

func Init(dir string, writer io.Writer) (created bool, err error) {
	created, err = initConfigFile(filepath.Join(dir, DefaultName))
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

	_, err = file.Write(DefaultConfig().Bytes())
	if err != nil {
		file.Close()
		return false, err
	}
	if err := file.Close(); err != nil {
		return false, err
	}
	return true, nil
}
