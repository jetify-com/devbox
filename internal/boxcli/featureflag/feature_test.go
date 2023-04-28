// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

import (
	"testing"

	"go.jetpack.io/devbox/internal/env"
)

func TestEnabledFeature(t *testing.T) {
	name := "TestEnabledFeature"
	enabled(name)
	if !features[name].Enabled() {
		t.Errorf("got %s.Enabled() = false, want true.", name)
	}
}

func TestDisabledFeature(t *testing.T) {
	name := "TestDisabledFeature"
	disabled(name)
	if features[name].Enabled() {
		t.Errorf("got %s.Enabled() = true, want false.", name)
	}
}

func TestEnabledFeatureEnv(t *testing.T) {
	name := "TestEnabledFeatureEnv"
	disabled(name)
	t.Setenv(env.DevboxFeaturePrefix+name, "1")
	if !features[name].Enabled() {
		t.Errorf("got %s.Enabled() = false, want true.", name)
	}
}

func TestNonExistentFeature(t *testing.T) {
	name := "TestNonExistentFeature"
	if features[name].Enabled() {
		t.Errorf("got %s.Enabled() = true, want false.", name)
	}
}
