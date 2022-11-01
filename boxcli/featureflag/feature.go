package featureflag

import (
	"os"
)

type feature struct {
	name    string
	enabled bool
}
type Feature interface {
	Enabled() bool
}

var features = map[string]Feature{}

func disabled(name string) {
	features[name] = &feature{name: name}
}

func enabled(name string) {
	features[name] = &feature{name: name, enabled: true}
}

func Get(name string) Feature {
	return features[name]
}

func (f *feature) Enabled() bool {
	if os.Getenv("DEVBOX_FEATURE_"+f.name) == "1" {
		return true
	}
	return f.enabled
}
