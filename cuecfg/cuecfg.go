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

func Marshal(valuePtr any, extension string) ([]byte, error) {
	err := cuego.Complete(valuePtr)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	switch extension {
	case ".json":
		return marshalJSON(valuePtr)
	case ".yml", ".yaml":
		return marshalYaml(valuePtr)
	case ".toml":
		return marshalToml(valuePtr)
	case ".xml", ".csproj":
		return marshalXML(valuePtr)
	}
	return nil, errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func Unmarshal(data []byte, extension string, valuePtr any) error {
	switch extension {
	case ".json":
		err := unmarshalJSON(data, valuePtr)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".yml", ".yaml":
		err := unmarshalYaml(data, valuePtr)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".toml":
		err := unmarshalToml(data, valuePtr)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	case ".xml", ".csproj":
		err := unmarshalXML(data, valuePtr)
		if err != nil {
			return errors.WithStack(err)
		}
		return nil
	}
	return errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func InitFile(path string, valuePtr any) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		// File already exists, don't create a new one.
		// TODO: should we read and write again, in case the schema needs updating?
		return false, nil
	} else if errors.Is(err, os.ErrNotExist) {
		// File does not exist, create a new one:
		return true, WriteFile(path, valuePtr)
	} else {
		// Error case:
		return false, errors.WithStack(err)
	}

}

func ParseFile(path string, valuePtr any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return Unmarshal(data, filepath.Ext(path), valuePtr)
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
