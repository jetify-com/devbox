// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"slices"
	"strings"
	"testing"
)

func TestWrapArgsForRun(t *testing.T) {
	tests := []struct {
		args string
		want string
	}{
		{"run hi --foo", "run hi -- --foo"},
		{"run -e A=1 --pure hi --foo", "run -e A=1 --pure hi -- --foo"},
		{"run hi --foo -- x", "run hi -- --foo -- x"},
		{"run hi -- --foo", "run hi -- --foo"},
		{"run -- hi --foo", "run -- hi --foo"},
		{"run hi", "run hi"},
		{"global run echo --foo", "global run echo -- --foo"},
		{"global run -e A=1 echo --foo", "global run -e A=1 echo -- --foo"},
		{"global add hello --foo", "global add hello --foo"},
		{"add hello --foo", "add hello --foo"},
	}
	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			got := wrapArgsForRun(RootCmd(), strings.Fields(tt.args))
			if want := strings.Fields(tt.want); !slices.Equal(got, want) {
				t.Errorf("wrapArgsForRun(%q) = %q, want %q", tt.args, got, want)
			}
		})
	}
}
