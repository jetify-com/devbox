// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
)

// Config defines a devbox environment as JSON.
type Config struct {
	// Packages is the slice of Nix packages that devbox makes available in
	// its environment.
	Packages []string `cue:"[...string]" json:"packages"`
	// InstallStage defines the actions that should be taken when
	// installing language-specific libraries.
	InstallStage *Stage `json:"install_stage,omitempty"`
	// BuildStage defines the actions that should be taken when
	// compiling the application binary.
	BuildStage *Stage `json:"build_stage,omitempty"`
	// StartStage defines the actions that should be taken when
	// starting (running) the application.
	StartStage *Stage `json:"start_stage,omitempty"`

	// Shell configures the devbox shell environment.
	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook string `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`
}

// This contains a subset of fields from plansdk.Stage
type Stage struct {
	Command string `cue:"string" json:"command"`
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
