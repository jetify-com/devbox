// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package configfile

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestSetStringField(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		field string
		val   string
		want  string
	}{
		{
			name:  "missing key with empty value stays missing",
			in:    `{ "packages": {} }`,
			field: "Name",
			val:   "",
			want:  `{ "packages": {} }`,
		},
		{
			name:  "missing key with value is added",
			in:    `{ "packages": {} }`,
			field: "Name",
			val:   "myproj",
			want:  `{ "packages": {}, "name": "myproj" }`,
		},
		{
			name:  "existing key with empty value is removed",
			in:    `{ "name": "old", "packages": {} }`,
			field: "Name",
			val:   "",
			want:  `{ "packages": {} }`,
		},
		{
			name:  "existing key with value is updated",
			in:    `{ "name": "old", "packages": {} }`,
			field: "Name",
			val:   "new",
			want:  `{ "packages": {}, "name": "new" }`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			c, err := LoadBytes([]byte(test.in))
			if err != nil {
				t.Fatalf("loading config: %v", err)
			}
			c.SetStringField(test.field, test.val)
			if diff := cmp.Diff([]byte(test.want), c.Bytes(), optParseHujson()); diff != "" {
				t.Errorf("config mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
