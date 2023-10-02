package envpath

import (
	"testing"
)

func TestCleanEnvPath(t *testing.T) {
	tests := []struct {
		name    string
		inPath  string
		outPath string
	}{
		{
			name:    "NoEmptyPaths",
			inPath:  "/usr/local/bin::",
			outPath: "/usr/local/bin",
		},
		{
			name:    "NoRelativePaths",
			inPath:  "/usr/local/bin:/usr/bin:../test:/bin:/usr/sbin:/sbin:.:..",
			outPath: "/usr/local/bin:/usr/bin:/bin:/usr/sbin:/sbin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := JoinPathLists(test.inPath)
			if got != test.outPath {
				t.Errorf("Got incorrect cleaned PATH.\ngot:  %s\nwant: %s", got, test.outPath)
			}
		})
	}
}
