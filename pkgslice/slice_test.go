// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.
package pkgslice

import (
	"fmt"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

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
			got := Exclude(tc.in, tc.exclude)
			if !slices.Equal(got, tc.out) {
				t.Errorf("Got slice %v, want %v.", got, tc.out)
			}
		})
	}
}

func TestUnique(t *testing.T) {
	cases := []struct{ in, out []string }{
		{
			in:  []string{},
			out: []string{},
		},
		{
			in:  []string{"a", "b", "c"},
			out: []string{"a", "b", "c"},
		},
		{
			in:  []string{"a", "c", "c"},
			out: []string{"a", "c"},
		},
		{
			in:  []string{"a", "a"},
			out: []string{"a"},
		},
	}

	for _, tc := range cases {
		name := strings.Join(tc.in, ",")

		t.Run(name, func(t *testing.T) {
			got := Unique(tc.in)
			if !slices.Equal(got, tc.out) {
				t.Errorf("Got slice %v, want %v.", got, tc.out)
			}
		})
	}
}
