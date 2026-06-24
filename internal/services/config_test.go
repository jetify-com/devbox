// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLookupProcessComposeOverride(t *testing.T) {
	tests := []struct {
		name         string
		files        []string
		wantFileName string
	}{
		{
			name:         "no override file",
			files:        []string{},
			wantFileName: "",
		},
		{
			name:         "yaml override",
			files:        []string{"process-compose.override.yaml"},
			wantFileName: "process-compose.override.yaml",
		},
		{
			name:         "yml override",
			files:        []string{"process-compose.override.yml"},
			wantFileName: "process-compose.override.yml",
		},
		{
			name:         "yaml takes precedence over yml",
			files:        []string{"process-compose.override.yaml", "process-compose.override.yml"},
			wantFileName: "process-compose.override.yaml",
		},
		{
			name:         "ignores non-override files",
			files:        []string{"process-compose.yaml", "other.yaml"},
			wantFileName: "",
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			// Create temp directory
			tempDir := t.TempDir()

			// Create test files
			for _, f := range testCase.files {
				path := filepath.Join(tempDir, f)
				if err := os.WriteFile(path, []byte("# test"), 0644); err != nil {
					t.Fatalf("Failed to create test file %s: %v", f, err)
				}
			}

			got := LookupProcessComposeOverride(tempDir)

			if testCase.wantFileName == "" {
				if got != "" {
					t.Errorf("LookupProcessComposeOverride() = %q, want empty string", got)
				}
			} else {
				want := filepath.Join(tempDir, testCase.wantFileName)
				if got != want {
					t.Errorf("LookupProcessComposeOverride() = %q, want %q", got, want)
				}
			}
		})
	}
}
