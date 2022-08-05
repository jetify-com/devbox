package cuecfg

import "encoding/json"

// TODO: consider using cue's JSON marshaller instead of
// "encoding/json" ... it might have extra functionality related
// to the cue language.
func MarshalJson(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func UnmarshalJson(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
