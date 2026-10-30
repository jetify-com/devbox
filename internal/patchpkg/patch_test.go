package patchpkg

import (
	"slices"
	"testing"
)

func TestSplitRPATH(t *testing.T) {
	tests := []struct {
		rpath string
		want  []string
	}{
		{rpath: "", want: []string{}},
		{rpath: "/a", want: []string{"/a"}},
		{rpath: "/a:/b", want: []string{"/a", "/b"}},
		{rpath: "/a::/b:", want: []string{"/a", "/b"}},
		{rpath: ":/a", want: []string{"/a"}},
	}
	for _, tt := range tests {
		got := splitRPATH(tt.rpath)
		if !slices.Equal(got, tt.want) {
			t.Errorf("splitRPATH(%q) = %q, want %q", tt.rpath, got, tt.want)
		}
	}
}
