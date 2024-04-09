// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"encoding/xml"
)

func marshalXML(v interface{}) ([]byte, error) {
	return xml.Marshal(v)
}

func unmarshalXML(data []byte, v interface{}) error {
	return xml.Unmarshal(data, v)
}
