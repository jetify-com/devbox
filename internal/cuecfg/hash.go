package cuecfg

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
)

func FileHash(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
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
