// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import "gopkg.in/yaml.v3"

// TODO: consider using cue's YAML marshaller.
// It might have extra functionality related
// to the cue language.
func marshalYaml(v interface{}) ([]byte, error) {
	return yaml.Marshal(v)
}

func unmarshalYaml(data []byte, v interface{}) error {
	return yaml.Unmarshal(data, v)
}
