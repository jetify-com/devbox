package plugin

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.jetpack.io/devbox/nix/flake"
)

func TestNewGitPlugin(t *testing.T) {
	testCases := []struct {
		name        string
		Include     []flake.Ref
		expected    gitPlugin
		expectedURL string
	}{
		{
			name: "parse basic github plugin",
			Include: []flake.Ref{
				{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "github",
					Host:  "github.com",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
				},
				name: "jetify-com.devbox-plugins",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/master",
		},
		{
			name: "parse github plugin with dir param",
			Include: []flake.Ref{
				{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "github",
					Host:  "github.com",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/master/mongodb",
		},
		{
			name: "parse github plugin with dir param and rev",
			Include: []flake.Ref{
				{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Ref:   "my-branch",
					Dir:   "mongodb",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "github",
					Host:  "github.com",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Ref:   "my-branch",
					Dir:   "mongodb",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/my-branch/mongodb",
		},
		{
			name: "parse github plugin with dir param and rev",
			Include: []flake.Ref{
				{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
					Ref:   "initials/my-branch",
					Rev:   "initials",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
					Ref:   "initials/my-branch", // FIXME
					Rev:   "initials",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/initials/my-branch/mongodb",
		},
		{
			name: "parse gitlab plugin",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-repo",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-repo",
				},
				name: "username.my-repo",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-repo/repository/files/raw?ref=main",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := newGitPlugin(testCase.Include[0]) // FIXME: need to evaluate URL
			assert.NoError(t, err)
			assert.Equal(t, &testCase.expected, actual)
			u, err := testCase.expected.url("")
			assert.Nil(t, err)
			assert.Equal(t, testCase.expectedURL, u)
		})
	}
}

func TestGitPluginAuth(t *testing.T) {
	gitPlugin := gitPlugin{
		ref: flake.Ref{
			Type:  "github",
			Owner: "jetpack-io",
			Repo:  "devbox-plugins",
		},
		name: "jetpack-io.devbox-plugins",
	}

	expectedURL := "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/master/test"

	t.Run("generate request for public Github repository", func(t *testing.T) {
		url, err := gitPlugin.url("test")
		assert.NoError(t, err)
		actual, err := gitPlugin.request(url)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, actual.URL.String())
		assert.Equal(t, "", actual.Header.Get("Authorization"))
	})

	t.Run("generate request for private Github repository", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "gh_abcd")
		url, err := gitPlugin.url("test")
		assert.NoError(t, err)
		actual, err := gitPlugin.request(url)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, actual.URL.String())
		assert.Equal(t, "token gh_abcd", actual.Header.Get("Authorization"))
	})
}
