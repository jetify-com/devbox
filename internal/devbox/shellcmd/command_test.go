// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package shellcmd

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"go.jetify.com/devbox/internal/cuecfg"
)

func TestCommandsUnmarshalString(t *testing.T) {
	tests := []struct {
		jsonIn string
		want   Commands
	}{
		{
			jsonIn: `null`,
			want: Commands{
				MarshalAs: CmdArray,
				Cmds:      nil,
			},
		},
		{
			jsonIn: `""`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{""},
			},
		},
		{
			jsonIn: `"\n"`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{"\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho 'line2'"`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo '\nline1'\necho 'line2'\n"`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{"echo '\nline1'\necho 'line2'\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\n\necho 'line2'"`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\n\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho '\tline2'"`,
			want: Commands{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho '\tline2'"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.jsonIn, func(t *testing.T) {
			got := Commands{}
			if err := json.Unmarshal([]byte(test.jsonIn), &got); err != nil {
				t.Fatal("Got error unmarshalling test input:", err)
			}
			if got.MarshalAs != test.want.MarshalAs {
				t.Errorf("Got MarshalAs == %s after unmarshalling, want "+
					"MarshalAs == %s.", got.MarshalAs, test.want.MarshalAs)
			}
			if len(got.Cmds) != len(test.want.Cmds) {
				t.Fatalf("len(got.Cmds) != len(want.Cmds)\ngot:  %q (len %d)\nwant: %q (len %d)",
					got.Cmds, len(got.Cmds), test.want.Cmds, len(test.want.Cmds))
			}
			for i := range got.Cmds {
				got, want := got.Cmds[i], test.want.Cmds[i]
				if got != want {
					t.Fatalf("got.Cmds[%[1]d] != want.Cmds[%[1]d]\ngot:  %q\nwant: %q",
						i, got, want)
				}
			}
			b, err := cuecfg.MarshalJSON(got)
			if err != nil {
				t.Fatal("Got error marshalling back to JSON:", err)
			}
			if diff := cmp.Diff(test.jsonIn, string(b)); diff != "" {
				t.Errorf("Got different JSON after unmarshalling and re-marshalling (-want +got):\n%s", diff)
			}
		})
	}
}

func TestCommandsString(t *testing.T) {
	tests := []struct {
		jsonIn string
		want   string
	}{
		{
			jsonIn: `null`,
			want:   "",
		},
		{
			jsonIn: `[]`,
			want:   "",
		},
		{
			jsonIn: `[""]`,
			want:   "",
		},
		{
			jsonIn: `["\n"]`,
			want:   "\n",
		},
		{
			jsonIn: `["echo 'line1'\necho 'line2'"]`,
			want:   "echo 'line1'\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'", "echo 'line2'"]`,
			want:   "echo 'line1'\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'\n\necho 'line2'"]`,
			want:   "echo 'line1'\n\necho 'line2'",
		},
		{
			jsonIn: `["echo 'line1'", "", "echo 'line2'"]`,
			want:   "echo 'line1'\n\necho 'line2'",
		},
	}
	for _, test := range tests {
		t.Run(test.jsonIn, func(t *testing.T) {
			got := Commands{}
			if err := json.Unmarshal([]byte(test.jsonIn), &got); err != nil {
				t.Fatal("Got error unmarshalling test input:", err)
			}
			if got.String() != test.want {
				t.Errorf("got.String() != want\ngot:  %q\nwant: %q",
					got.String(), test.want)
			}
		})
	}
}

func ExampleCommands_AppendScript() {
	shCmds := Commands{}
	shCmds.AppendScript(`
		# This script will be unindented by the number of leading tabs
		# on the first line.
		if true; then
			echo "this is always printed"
		fi`,
	)
	b, _ := cuecfg.MarshalJSON(&shCmds)
	fmt.Println(string(b))

	// Output:
	// [
	//   "# This script will be unindented by the number of leading tabs",
	//   "# on the first line.",
	//   "if true; then",
	//   "\techo \"this is always printed\"",
	//   "fi"
	// ]
}

func TestAppendScript(t *testing.T) {
	tests := []struct {
		name     string
		script   string
		wantCmds []string
	}{
		{
			name:     "Empty",
			script:   "",
			wantCmds: nil,
		},
		{
			name:     "OnlySpaces",
			script:   " ",
			wantCmds: nil,
		},
		{
			name:     "Only newlines",
			script:   "\r\n",
			wantCmds: nil,
		},
		{
			name:     "Simple",
			script:   "echo test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "LeadingNewline",
			script:   "\necho test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "LeadingNewlineAndSpace",
			script:   "\n    echo test",
			wantCmds: []string{"echo test"},
		},
		{
			name:     "TrailingWhitespace",
			script:   "echo test  \n",
			wantCmds: []string{"echo test"},
		},
		{
			name:   "SecondLineIndent",
			script: "if true; then\n\techo test\nfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
		{
			name:   "Unindent",
			script: "\n\tif true; then\n\t\techo test\n\tfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
		{
			name:   "UnindentTooFewTabs",
			script: "\t\tif true; then\n\techo test\n\t\tfi",
			wantCmds: []string{
				"if true; then",
				"\techo test",
				"fi",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			shCmds := Commands{}
			shCmds.AppendScript(test.script)
			gotCmds := shCmds.Cmds
			if diff := cmp.Diff(test.wantCmds, gotCmds); diff != "" {
				t.Errorf("Got incorrect commands slice (-want +got):\n%s", diff)
			}
		})
	}
}
