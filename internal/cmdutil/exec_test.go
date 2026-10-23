// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package cmdutil

import (
	"strings"
	"testing"
)

func TestCommandTTYWithBufferCapturesStderr(t *testing.T) {
	cmd, buf := CommandTTYWithBuffer("sh", "-c", "echo to-stdout; echo to-stderr >&2")
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"to-stdout", "to-stderr"} {
		if !strings.Contains(buf.String(), want) {
			t.Errorf("buffer %q does not contain %q", buf.String(), want)
		}
	}
}
