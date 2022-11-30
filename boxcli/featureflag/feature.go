package featureflag

import (
	"os"
	"strconv"

	"go.jetpack.io/devbox/debug"
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

func (f *feature) Enabled() bool {
	if f == nil {
		return false
	}
	if on, err := strconv.ParseBool(os.Getenv("DEVBOX_FEATURE_" + f.name)); err == nil {
		status := "enabled"
		if !on {
			status = "disabled"
		}
		debug.Log("Feature %q %s via environment variable.", f.name, status)
		return on
	}
	return f.enabled
}
