// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package mutagen

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeSessionName(t *testing.T) {
	testCases := []struct {
		input     string
		sanitized string
	}{
		{"7foo", "a7foo"},
		{"foo", "foo"},
		{"foo/bar", "foo-bar"},
		{"foo/bar/baz", "foo-bar-baz"},
		{"foo.bar", "foo-bar"},
		{"foo_bar", "foo-bar"},
	}

	for _, testCase := range testCases {
		t.Run(testCase.input, func(t *testing.T) {
			assert := assert.New(t)
			result := SanitizeSessionName(testCase.input)
			assert.Equal(testCase.sanitized, result)
		})
	}
}
