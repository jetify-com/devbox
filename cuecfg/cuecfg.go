// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"os"
	"path/filepath"

	"cuelang.org/go/cuego"
	"github.com/pkg/errors"
)

// TODO: add support for .cue

func Marshal(value any, extension string) ([]byte, error) {
	err := cuego.Complete(value)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch extension {
	case ".json":
		return MarshalJSON(value)
	case ".yml", ".yaml":
		return MarshalYaml(value)
	case ".toml":
		return MarshalToml(value)
	}
	return nil, errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func Unmarshal(data []byte, extension string, value any) error {
	switch extension {
	case ".json":
		err := UnmarshalJSON(data, value)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".yml", ".yaml":
		err := UnmarshalYaml(data, value)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".toml":
		err := UnmarshalToml(data, value)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	return errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func InitFile(path string, value any) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		// File already exists, don't create a new one.
		// TODO: should we read and write again, in case the schema needs updating?
		return false, nil
	} else if errors.Is(err, os.ErrNotExist) {
		// File does not exist, create a new one:
		return true, WriteFile(path, value)
	} else {
		// Error case:
		return false, errors.WithStack(err)
	}

}

func ReadFile(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return Unmarshal(data, filepath.Ext(path), value)
}

func WriteFile(path string, value any) error {
	data, err := Marshal(value, filepath.Ext(path))
	if err != nil {
		return errors.WithStack(err)
	}

	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
