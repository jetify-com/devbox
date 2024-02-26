package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/nix/flake"
)

func TestNewGithubPlugin(t *testing.T) {
	testCases := []struct {
		name     string
		Include  string
		expected githubPlugin
	}{
		{
			name:    "parse basic github plugin",
			Include: "github:jetpack-io/devbox-plugins",
			expected: githubPlugin{
				RefLike: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "master",
					},
					filename: pluginConfigName,
				},
			},
		},
		{
			name:    "parse github plugin with dir param",
			Include: "github:jetpack-io/devbox-plugins?dir=mongodb",
			expected: githubPlugin{
				RefLike: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "master",
						Dir:   "mongodb",
					},
					filename: pluginConfigName,
				},
			},
		},
		{
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetpack-io/devbox-plugins/my-branch?dir=mongodb",
			expected: githubPlugin{
				RefLike: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "my-branch",
						Dir:   "mongodb",
					},
					filename: pluginConfigName,
				},
			},
		},
		{
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetpack-io/devbox-plugins/initials/my-branch?dir=mongodb",
			expected: githubPlugin{
				RefLike: RefLike{
					Ref: flake.Ref{
						Type:  "github",
						Owner: "jetpack-io",
						Repo:  "devbox-plugins",
						Ref:   "initials/my-branch",
						Dir:   "mongodb",
					},
					filename: pluginConfigName,
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, _ := parseReflike(testCase.Include)
			assert.Equal(t, &testCase.expected, actual)
		})
	}
}
