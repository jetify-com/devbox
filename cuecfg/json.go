// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"bytes"
	"encoding/json"
)

// TODO: consider using cue's JSON marshaller instead of
// "encoding/json" ... it might have extra functionality related
// to the cue language.
func marshalJSON(v interface{}) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func unmarshalJSON(data []byte, v interface{}) error {
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	return dec.Decode(v)
}
