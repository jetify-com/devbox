// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

// TODO: add support for .cue

func Marshal(valuePtr any, extension string) ([]byte, error) {
	switch extension {
	case ".json", ".lock":
		return MarshalJSON(valuePtr)
	case ".yml", ".yaml":
		return marshalYaml(valuePtr)
	case ".toml":
		return marshalToml(valuePtr)
	case ".xml":
		return marshalXML(valuePtr)
	}
	return nil, errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func Unmarshal(data []byte, extension string, valuePtr any) error {
	switch extension {
	case ".json", ".lock":
		return errors.WithStack(unmarshalJSON(data, valuePtr))
	case ".yml", ".yaml":
		return errors.WithStack(unmarshalYaml(data, valuePtr))
	case ".toml":
		return errors.WithStack(unmarshalToml(data, valuePtr))
	case ".xml":
		return errors.WithStack(unmarshalXML(data, valuePtr))
	}
	return errors.Errorf("Unsupported file format '%s' for config file", extension)
}

func InitFile(path string, valuePtr any) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		// File already exists, don't create a new one.
		// TODO: should we read and write again, in case the schema needs updating?
		return false, nil
	}
	if errors.Is(err, fs.ErrNotExist) {
		// File does not exist, create a new one:
		return true, WriteFile(path, valuePtr)
	}
	// Error case:
	return false, errors.WithStack(err)
}

func ParseFile(path string, valuePtr any) error {
	return ParseFileWithExtension(path, filepath.Ext(path), valuePtr)
}

// ParseFileWithExtension lets the caller override the extension of the `path` filename
// For example, project.csproj files should be treated as having extension .xml
func ParseFileWithExtension(path, ext string, valuePtr any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return errors.WithStack(err)
	}

	return Unmarshal(data, ext, valuePtr)
}

func WriteFile(path string, value any) error {
	data, err := Marshal(value, filepath.Ext(path))
	if err != nil {
		return errors.WithStack(err)
	}
	data = append(data, '\n')
	return errors.WithStack(os.WriteFile(path, data, 0o644))
}

func IsSupportedExtension(ext string) bool {
	switch ext {
	case ".json", ".lock", ".yml", ".yaml", ".toml", ".xml":
		return true
	default:
		return false
	}
}
