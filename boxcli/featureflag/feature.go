package featureflag

import (
	"os"
)

type feature struct {
	name    string
	enabled bool
}

var features = map[string]*feature{}

func disabled(name string) {
	features[name] = &feature{name: name}
}

func enabled(name string) {
	features[name] = &feature{name: name, enabled: true}
}

func Get(name string) *feature {
	return features[name]
}

func (f *feature) Enabled() bool {
	if f == nil {
		return false
	}
	if os.Getenv("DEVBOX_FEATURE_"+f.name) == "1" {
		return true
	}
	return f.enabled
}
