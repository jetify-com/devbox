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
	case "json":
		return MarshalJson(v)
	case "yml", "yaml":
		return MarshalYaml(v)
	}
	return nil, errors.New("unsupported extension")
}

func Unmarshal(data []byte, extension string, v any) error {
	switch extension {
	case "json":
		err := UnmarshalJson(data, v)
		if err != nil {
			return errors.WithStack(err)
		}
	case "yml", "yaml":
		err := UnmarshalYaml(data, v)
		if err != nil {
			return errors.WithStack(err)
		}
	}
	return nil
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
