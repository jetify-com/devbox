// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"os"
	"path/filepath"

	"cuelang.org/go/cuego"
	"github.com/pkg/errors"
)

// TODO: add support for .cue and possible .toml

func Marshal(v any, extension string) ([]byte, error) {
	err := cuego.Complete(v)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch extension {
	case ".json":
		return MarshalJSON(v)
	case ".yml", ".yaml":
		return MarshalYaml(v)
	}
	return nil, errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func Unmarshal(data []byte, extension string, v any) error {
	switch extension {
	case ".json":
		err := UnmarshalJSON(data, v)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".yml", ".yaml":
		err := UnmarshalYaml(data, v)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	return errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func InitFile(path string, v any) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		// File already exists, don't create a new one.
		// TODO: should we read and write again, in case the schema needs updating?
		return false, nil
	} else if errors.Is(err, os.ErrNotExist) {
		// File does not exist, create a new one:
		return true, WriteFile(path, v)
	} else {
		// Error case:
		return false, errors.WithStack(err)
	}

}

func ReadFile(path string, v any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return Unmarshal(data, filepath.Ext(path), v)
}

func WriteFile(path string, v any) error {
	data, err := Marshal(v, filepath.Ext(path))
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
