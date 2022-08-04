package devbox

import (
	"encoding/json"
	"os"
	"path/filepath"

	"cuelang.org/go/cuego"
	"github.com/pkg/errors"
)

type DevConfig struct {
	Packages []string `cue:"[...string]" json:"packages,omitempty"`
}

func LoadDevConfig(path string) *DevConfig {
	cfgPath := filepath.Join(path, "devbox.json")

	if _, err := os.Stat(cfgPath); os.IsNotExist(err) {
		return &DevConfig{}
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		return &DevConfig{}
	}
	return cfg
}

func Load(path string) (*DevConfig, error) {
	// Load the data from the file
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Parse it into the go struct
	devCfg := &DevConfig{}
	err = json.Unmarshal(data, devCfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// Complete and validate using CUE
	err = cuego.Complete(devCfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	return devCfg, nil
}

func Write(path string, config *DevConfig) error {
	// Ensure the data validates before writting
	err := cuego.Validate(config)
	if err != nil {
		return errors.WithStack(err)
	}

	// Convert to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}

	// Write the file
	err = os.WriteFile(path, data, 0644)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}
