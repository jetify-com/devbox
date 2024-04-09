// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cloud

import (
	"testing"
)

func TestParseUsernameFromErrorMessage(t *testing.T) {
	testCases := []struct {
		name       string
		errMessage string
		username   string
	}{
		{
			"success_case",
			"Hi myDearUsername! You've successfully authenticated, but GitHub does not provide shell access.",
			"myDearUsername",
		},
		{
			// NOTE this case won't actually occur because parseUsernameFromErrorMessage
			// is only run for ExitCode == 1, but pub_key_denied is a local ssh error with ExitCode == 255
			//
			// Adding the test case to exercise the scenario where the error message doesn't match the regexp.
			"pub_key_denied_case",
			"public key denied",
			"",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			result := parseUsernameFromErrorMessage(testCase.errMessage)
			if result != testCase.username {
				t.Errorf("expected %s username but got %s username", testCase.username, result)
			}
		})
	}
}
