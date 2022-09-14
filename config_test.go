package devbox

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestConfigShellCmdsUnmarshalString(t *testing.T) {
	tests := []struct {
		jsonIn string
		want   ConfigShellCmds
	}{
		{
			jsonIn: `null`,
			want: ConfigShellCmds{
				MarshalAs: CmdArray,
				Cmds:      nil,
			},
		},
		{
			jsonIn: `""`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{""},
			},
		},
		{
			jsonIn: `"\n"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho 'line2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo '\nline1'\necho 'line2'\n"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo '\nline1'\necho 'line2'\n"},
			},
		},
		{
			jsonIn: `"echo 'line1'\n\necho 'line2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\n\necho 'line2'"},
			},
		},
		{
			jsonIn: `"echo 'line1'\necho '\tline2'"`,
			want: ConfigShellCmds{
				MarshalAs: CmdString,
				Cmds:      []string{"echo 'line1'\necho '\tline2'"},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.jsonIn, func(t *testing.T) {
			got := ConfigShellCmds{}
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
			b, err := json.Marshal(got)
			if err != nil {
				t.Fatal("Got error marshalling back to JSON:", err)
			}
			if diff := cmp.Diff(test.jsonIn, string(b)); diff != "" {
				t.Errorf("Got different JSON after unmarshalling and re-marshalling (-want +got):\n%s", diff)
			}
		})
	}
}

func TestConfigShellCmdsString(t *testing.T) {
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
			got := ConfigShellCmds{}
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
