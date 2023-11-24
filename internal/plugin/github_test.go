package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewGithubPlugin(t *testing.T) {
	testCases := []struct {
		name     string
		expected githubPlugin
	}{
		{
			name: "parse basic github plugin",
			expected: githubPlugin{
				raw:      "jetpack-io/devbox-plugins",
				org:      "jetpack-io",
				repo:     "devbox-plugins",
				revision: "master",
			},
		},
		{
			name: "parse github plugin with dir param",
			expected: githubPlugin{
				raw:      "jetpack-io/devbox-plugins?dir=mongodb",
				org:      "jetpack-io",
				repo:     "devbox-plugins",
				revision: "master",
				dir:      "mongodb",
			},
		},
		{
			name: "parse github plugin with dir param and rev",
			expected: githubPlugin{
				raw:      "jetpack-io/devbox-plugins/my-branch?dir=mongodb",
				org:      "jetpack-io",
				repo:     "devbox-plugins",
				revision: "my-branch",
				dir:      "mongodb",
			},
		},
		{
			name: "parse github plugin with dir param and rev",
			expected: githubPlugin{
				raw:      "jetpack-io/devbox-plugins/initials/my-branch?dir=mongodb",
				org:      "jetpack-io",
				repo:     "devbox-plugins",
				revision: "initials/my-branch",
				dir:      "mongodb",
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, _ := newGithubPlugin(testCase.expected.raw)
			assert.Equal(t, actual, &testCase.expected)
		})
	}
}
