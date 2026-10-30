// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package pkgdef

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMatchVersion(t *testing.T) {
	available := []string{"1.21.0", "1.22.10", "1.22.3", "1.22.9", "1.23.0-rc1", "1.21rc2", "1.23"}

	tests := []struct {
		requested string
		want      string
	}{
		{"latest", "1.23"},
		{"", "1.23"},
		{"1.22", "1.22.10"},
		{"1.22.3", "1.22.3"},
		{"1.21", "1.21.0"},
		{"1.23", "1.23"},
		{"1.23.0-rc1", "1.23.0-rc1"}, // pre-releases only match exactly
		{"1.21rc2", "1.21rc2"},
	}
	for _, tt := range tests {
		t.Run(tt.requested, func(t *testing.T) {
			got, err := MatchVersion(available, tt.requested)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMatchVersionNotFound(t *testing.T) {
	_, err := MatchVersion([]string{"1.22.3"}, "1.24")
	require.ErrorIs(t, err, ErrVersionNotFound)
	assert.Contains(t, err.Error(), "1.22.3")

	_, err = MatchVersion(nil, "latest")
	require.ErrorIs(t, err, ErrVersionNotFound)

	// "1.2" must not match "1.22.3" by string prefix.
	_, err = MatchVersion([]string{"1.22.3"}, "1.2")
	require.ErrorIs(t, err, ErrVersionNotFound)

	// Only pre-releases available: "latest" has nothing to pick.
	_, err = MatchVersion([]string{"1.0.0-beta"}, "latest")
	require.ErrorIs(t, err, ErrVersionNotFound)
}

func TestCompareVersions(t *testing.T) {
	assert.Negative(t, CompareVersions("1.9", "1.10"))
	assert.Positive(t, CompareVersions("1.22.10", "1.22.9"))
	assert.Negative(t, CompareVersions("1.22", "1.22.1"))
	assert.Equal(t, 0, CompareVersions("20.11.1", "20.11.1"))
	assert.Negative(t, CompareVersions("1.0a", "1.0b"))
}

func TestSortVersions(t *testing.T) {
	versions := []string{"1.9", "1.10", "2.0", "1.10.1"}
	SortVersions(versions)
	assert.Equal(t, []string{"2.0", "1.10.1", "1.10", "1.9"}, versions)
}

func TestIsPrerelease(t *testing.T) {
	assert.True(t, IsPrerelease("1.0.0-rc1"))
	assert.True(t, IsPrerelease("1.21rc2"))
	assert.True(t, IsPrerelease("2.0.0beta"))
	assert.False(t, IsPrerelease("1.22.3"))
	assert.False(t, IsPrerelease("20"))
}

func TestSummarize(t *testing.T) {
	assert.Equal(t, "none", summarize(nil))
	assert.Equal(t, "2.0, 1.0", summarize([]string{"1.0", "2.0"}))
	assert.Equal(t, "7, 6, 5, 4, 3, and 2 more", summarize([]string{"1", "2", "3", "4", "5", "6", "7"}))
}
