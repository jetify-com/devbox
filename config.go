// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"github.com/pkg/errors"
	"go.jetpack.io/axiom/opensource/devbox/cuecfg"
)

type Config struct {
	Packages []string `cue:"[...string]" json:"packages,omitempty"`
}

func ReadConfig(path string) (*Config, error) {
	cfg := &Config{}
	err := cuecfg.ReadFile(path, cfg)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

func WriteConfig(path string, cfg *Config) error {
	return cuecfg.WriteFile(path, cfg)
}
