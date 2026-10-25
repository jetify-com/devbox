// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import "testing"

func TestGenerateSubcommandsAcceptConfigFlags(t *testing.T) {
	for _, sub := range generateCmd().Commands() {
		t.Run(sub.Name(), func(t *testing.T) {
			if err := sub.ParseFlags([]string{"-c", "./proj", "--environment", "prod"}); err != nil {
				t.Errorf("generate %s: %v", sub.Name(), err)
			}
		})
	}
}
