// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"github.com/pelletier/go-toml/v2"
)

// TODO: consider using cue's JSON marshaller instead of
// "encoding/json" ... it might have extra functionality related
// to the cue language.
func MarshalToml(v interface{}) ([]byte, error) {
	return toml.Marshal(v)
}

func UnmarshalToml(data []byte, v interface{}) error {
	return toml.Unmarshal(data, v)
}
