package featureflag

import "testing"

func TestEnabledFeature(t *testing.T) {
	enabled("TEST")
	if !Get("TEST").Enabled() {
		t.Errorf("got Get(%q).Enabled() = false, want true.", "TEST")
	}
}

func TestDisabledFeature(t *testing.T) {
	disabled("TEST")
	if Get("TEST").Enabled() {
		t.Errorf("got Get(%q).Enabled() = true, want false.", "TEST")
	}
}

func TestEnabledFeatureEnv(t *testing.T) {
	disabled("TEST")
	t.Setenv("DEVBOX_FEATURE_TEST", "1")
	if !Get("TEST").Enabled() {
		t.Errorf("got Get(%q).Enabled() = false, want true.", "TEST")
	}
}

func TestNonExistentFeature(t *testing.T) {
	if Get("TEST").Enabled() {
		t.Errorf("got Get(%q).Enabled() = true, want false.", "TEST")
	}
}
