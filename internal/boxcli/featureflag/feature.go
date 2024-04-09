// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

import (
	"os"
	"strconv"
	"testing"

	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/envir"
)

type feature struct {
	name    string
	enabled bool
}

var features = map[string]*feature{}

// Prevent lint complaining about unused function
//
//nolint:unparam
func disable(name string) *feature {
	if features[name] == nil {
		features[name] = &feature{name: name}
	}
	features[name].enabled = false
	return features[name]
}

// Prevent lint complaining about unused function
//
//nolint:unparam
func enable(name string) *feature {
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
	if on, err := strconv.ParseBool(os.Getenv(envir.DevboxFeaturePrefix + f.name)); err == nil {
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

func (f *feature) EnableOnDev() *feature {
	if build.IsDev {
		f.enabled = true
	}
	return f
}

func (f *feature) EnableForTest(t *testing.T) {
	t.Setenv(envir.DevboxFeaturePrefix+f.name, "1")
}

// All returns a map of all known features flags and whether they're enabled.
func All() map[string]bool {
	m := make(map[string]bool, len(features))
	for name, feat := range features {
		m[name] = feat.Enabled()
	}
	return m
}
