// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cuecfg

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io/fs"
	"os"
)

func FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func Hash(s any) (string, error) {
	json, err := MarshalJSON(s)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(json)
	return hex.EncodeToString(hash[:]), nil
}
