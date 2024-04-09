// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"github.com/pelletier/go-toml/v2"
)

func marshalToml(v interface{}) ([]byte, error) {
	return toml.Marshal(v)
}

func unmarshalToml(data []byte, v interface{}) error {
	return toml.Unmarshal(data, v)
}
