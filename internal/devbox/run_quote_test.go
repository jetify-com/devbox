// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package devbox

import (
	"os"
	"os/exec"
	"strings"
	"testing"
)

// TestRunArgsKeepNewlinesAndTabs is the regression for devbox run turning a
// newline or tab inside an argument into the two characters \n or \t.
// It builds DEVBOX_RUN_CMD the same way RunScript does and evals it with sh,
// which is what nix.RunScript executes. Variable expansion stays on purpose.
func TestRunArgsKeepNewlinesAndTabs(t *testing.T) {
	cases := []struct {
		name string
		arg  string
		env  string
		want string
	}{
		{name: "newline", arg: "a\nb", want: "a\nb"},
		{name: "tab", arg: "a\tb", want: "a\tb"},
		{name: "backslash-n", arg: `a\nb`, want: `a\nb`},
		{name: "spaces", arg: "c d", want: "c d"},
		{name: "quotes", arg: `he said "hello"`, want: `he said "hello"`},
		{
			name: "variable",
			arg:  "$DEVBOX_QUOTE_TEST_VAR",
			env:  "DEVBOX_QUOTE_TEST_VAR=expanded",
			want: "expanded",
		},
	}
	for _, tcase := range cases {
		t.Run(tcase.name, func(t *testing.T) {
			envCommand := strings.Join([]string{"printf '%s'", quoteRunArg(tcase.arg)}, " ")
			cmd := exec.Command("sh", "-c", runCmdEval)
			cmd.Env = append(os.Environ(), "DEVBOX_RUN_CMD="+envCommand)
			if tcase.env != "" {
				cmd.Env = append(cmd.Env, tcase.env)
			}
			got, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("sh: %v\n%s", err, got)
			}
			if string(got) != tcase.want {
				t.Fatalf("got %q want %q", got, tcase.want)
			}
		})
	}
}
