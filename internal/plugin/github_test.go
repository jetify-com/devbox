package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/nix/flake"
)

func TestNewGithubPlugin(t *testing.T) {
	testCases := []struct {
		name        string
		Include     string
		expected    githubPlugin
		expectedURL string
	}{
		{
			name:    "parse basic github plugin",
			Include: "github:jetpack-io/devbox-plugins",
			expected: githubPlugin{
				ref: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
					},
					raw:      "github:jetpack-io/devbox-plugins",
					filename: pluginConfigName,
				},
			},
			expectedURL: "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/master",
		},
		{
			name:    "parse github plugin with dir param",
			Include: "github:jetpack-io/devbox-plugins?dir=mongodb",
			expected: githubPlugin{
				ref: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Dir:   "mongodb",
					},
					raw:      "github:jetpack-io/devbox-plugins?dir=mongodb",
					filename: pluginConfigName,
				},
			},
			expectedURL: "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/master/mongodb",
		},
		{
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetpack-io/devbox-plugins/my-branch?dir=mongodb",
			expected: githubPlugin{
				ref: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "my-branch",
						Dir:   "mongodb",
					},
					raw:      "github:jetpack-io/devbox-plugins/my-branch?dir=mongodb",
					filename: pluginConfigName,
				},
			},
			expectedURL: "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/my-branch/mongodb",
		},
		{
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetpack-io/devbox-plugins/initials/my-branch?dir=mongodb",
			expected: githubPlugin{
				ref: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "initials/my-branch",
						Dir:   "mongodb",
					},
					raw:      "github:jetpack-io/devbox-plugins/initials/my-branch?dir=mongodb",
					filename: pluginConfigName,
				},
			},
			expectedURL: "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/initials/my-branch/mongodb",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, _ := parseReflike(testCase.Include, "")
			assert.Equal(t, &testCase.expected, actual)
			u, err := testCase.expected.url("")
			assert.Nil(t, err)
			assert.Equal(t, testCase.expectedURL, u)
		})
	}
}
