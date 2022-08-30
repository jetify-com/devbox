package devbox

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

func TestUnique(t *testing.T) {
	cases := []struct{ in, out []string }{
		{
			in:  []string{"a", "b", "b", "c"},
			out: []string{"a", "b", "c"},
		},
		{
			in:  []string{},
			out: []string{},
		},
		{
			in:  []string{"a", "b", "c"},
			out: []string{"a", "b", "c"},
		},
		{
			in:  []string{"a", "a"},
			out: []string{"a"},
		},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("{%s}", strings.Join(tc.in, ",")), func(t *testing.T) {
			got := unique(tc.in)
			if !slices.Equal(got, tc.out) {
				t.Errorf("Got slice %v, want %v.", got, tc.out)
			}
		})
	}
}

func TestExclude(t *testing.T) {
	cases := []struct{ in, exclude, out []string }{
		{
			in:      []string{},
			exclude: []string{},
			out:     []string{},
		},
		{
			in:      []string{},
			exclude: []string{"a"},
			out:     []string{},
		},
		{
			in:      []string{"a"},
			exclude: []string{"a"},
			out:     []string{},
		},
		{
			in:      []string{"a", "b", "c"},
			exclude: []string{"b"},
			out:     []string{"a", "c"},
		},
		{
			in:      []string{"a", "b", "c"},
			exclude: []string{"a", "b"},
			out:     []string{"c"},
		},
		{
			in:      []string{"a", "b", "c"},
			exclude: []string{"a", "d"},
			out:     []string{"b", "c"},
		},
	}

	for _, tc := range cases {
		name := fmt.Sprintf("{%s}-{%s}",
			strings.Join(tc.in, ","),
			strings.Join(tc.exclude, ","))

		t.Run(name, func(t *testing.T) {
			got := exclude(tc.in, tc.exclude)
			if !slices.Equal(got, tc.out) {
				t.Errorf("Got slice %v, want %v.", got, tc.out)
			}
		})
	}
}
