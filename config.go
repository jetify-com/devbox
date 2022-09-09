// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

// Config defines a devbox environment as JSON.
type Config struct {
	plansdk.SharedPlan

	// Packages is the slice of Nix packages that devbox makes available in
	// its environment.
	Packages []string `cue:"[...string]" json:"packages"`

	// Shell configures the devbox shell environment.
	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook string `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`
}

// ReadConfig reads a devbox config file.
func ReadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := cuecfg.ParseFile(path, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

// WriteConfig saves a devbox config file.
func WriteConfig(path string, cfg *Config) error {
	return cuecfg.WriteFile(path, cfg)
}
