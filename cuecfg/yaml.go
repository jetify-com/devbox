package cuecfg

import "gopkg.in/yaml.v3"

// TODO: consider using cue's YAML marshaller.
// It might have extra functionality related
// to the cue language.
func MarshalYaml(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func UnmarshalYaml(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
