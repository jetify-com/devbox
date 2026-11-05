// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import "testing"

func TestDoLogCommand(t *testing.T) {
	tests := []struct {
		args    []string
		wantErr bool
	}{
		{args: []string{"shell-ready", "1700000000"}},
		{args: []string{"shell-interactive", "1700000000"}},
		{args: []string{"shell-ready"}, wantErr: true},
		{args: []string{"unknown-event", "1700000000"}, wantErr: true},
		{args: []string{}, wantErr: true},
	}
	for _, tt := range tests {
		err := doLogCommand(logCmd(), tt.args)
		if (err != nil) != tt.wantErr {
			t.Errorf("doLogCommand(%q) error = %v, wantErr %v", tt.args, err, tt.wantErr)
		}
	}
}
