// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"bytes"
	"encoding/json"

	"github.com/pkg/errors"
)

const Indent = "  "

// MarshalJSON marshals the given value to JSON. It does not HTML escape and
// adds standard indentation.
//
// TODO: consider using cue's JSON marshaller instead of
// "encoding/json" ... it might have extra functionality related
// to the cue language.
func MarshalJSON(v interface{}) ([]byte, error) {
	buff := &bytes.Buffer{}
	e := json.NewEncoder(buff)
	e.SetIndent("", Indent)
	e.SetEscapeHTML(false)
	if err := e.Encode(v); err != nil {
		return nil, errors.WithStack(err)
	}
	return bytes.TrimRight(buff.Bytes(), "\n"), nil
}

func unmarshalJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
