package nix

import (
	"fmt"
	"testing"
)

func TestContentAddressedPath(t *testing.T) {

	testCases := []struct {
		storePath string
		expected  string
	}{
		{
			"/nix/store/r2jd6ygnmirm2g803mksqqjm4y39yi6i-git-2.33.1",
			"/nix/store/ldbhlwhh39wha58rm61bkiiwm6j7211j-git-2.33.1",
		},
	}

	for index, testCase := range testCases {
		t.Run(fmt.Sprintf("%d", index), func(t *testing.T) {
			out, err := ContentAddressedStorePath(testCase.storePath)
			if err != nil {
				t.Errorf("got error: %v", err)
			}
			if out != testCase.expected {
				t.Errorf("got %s, want %s", out, testCase.expected)
			}
		})

	}
}
