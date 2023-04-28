// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

import (
	"os"
	"strconv"

	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/env"
)

type feature struct {
	name    string
	enabled bool
}

var features = map[string]*feature{}

func disabled(name string) *feature {
	if features[name] == nil {
		features[name] = &feature{name: name}
	}
	features[name].enabled = false
	return features[name]
}

func enabled(name string) *feature {
	if features[name] == nil {
		features[name] = &feature{name: name}
	}
	features[name].enabled = true
	return features[name]
}

var logMap = map[string]bool{}

func (f *feature) Enabled() bool {
	if f == nil {
		return false
	}
	if on, err := strconv.ParseBool(os.Getenv(env.DevboxFeaturePrefix + f.name)); err == nil {
		status := "enabled"
		if !on {
			status = "disabled"
		}
		if !logMap[f.name] {
			debug.Log("Feature %q %s via environment variable.", f.name, status)
			logMap[f.name] = true
		}
		return on
	}
	return f.enabled
}

func (f *feature) Disabled() bool {
	return !f.Enabled()
}

// All returns a map of all known features flags and whether they're enabled.
func All() map[string]bool {
	m := make(map[string]bool, len(features))
	for name, feat := range features {
		m[name] = feat.Enabled()
	}
	return m
}
