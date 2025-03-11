package plugin

import (
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"go.jetify.com/devbox/nix/flake"
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
			Include: "github:jetify-com/devbox-plugins",
			expected: githubPlugin{
				ref: flake.Ref{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
				},
				name: "jetify-com.devbox-plugins",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/master",
		},
		{
			name:    "parse github plugin with dir param",
			Include: "github:jetify-com/devbox-plugins?dir=mongodb",
			expected: githubPlugin{
				ref: flake.Ref{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/master/mongodb",
		},
		{
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetify-com/devbox-plugins/my-branch?dir=mongodb",
			expected: githubPlugin{
				ref: flake.Ref{
					Type:  "github",
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
			name:    "parse github plugin with dir param and rev",
			Include: "github:jetify-com/devbox-plugins/initials/my-branch?dir=mongodb",
			expected: githubPlugin{
				ref: flake.Ref{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Ref:   "initials/my-branch",
					Dir:   "mongodb",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/initials/my-branch/mongodb",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := newGithubPluginForTest(testCase.Include)
			assert.NoError(t, err)
			assert.Equal(t, &testCase.expected, actual)
			u, err := testCase.expected.url("")
			assert.Nil(t, err)
			assert.Equal(t, testCase.expectedURL, u)
		})
	}
}

// keep in sync with newGithubPlugin
func newGithubPluginForTest(include string) (*githubPlugin, error) {
	ref, err := flake.ParseRef(include)
	if err != nil {
		return nil, err
	}

	plugin := &githubPlugin{ref: ref}
	name := strings.ReplaceAll(ref.Dir, "/", "-")
	plugin.name = githubNameRegexp.ReplaceAllString(
		strings.Join(lo.Compact([]string{ref.Owner, ref.Repo, name}), "."),
		" ",
	)
	return plugin, nil
}

func TestGithubPluginAuth(t *testing.T) {
	githubPlugin := githubPlugin{
		ref: flake.Ref{
			Type:  "github",
			Owner: "jetpack-io",
			Repo:  "devbox-plugins",
		},
		name: "jetpack-io.devbox-plugins",
	}

	expectedURL := "https://raw.githubusercontent.com/jetpack-io/devbox-plugins/master/test"

	t.Run("generate request for public Github repository", func(t *testing.T) {
		url, err := githubPlugin.url("test")
		assert.NoError(t, err)
		actual, err := githubPlugin.request(url)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, actual.URL.String())
		assert.Equal(t, "", actual.Header.Get("Authorization"))
	})

	t.Run("generate request for private Github repository", func(t *testing.T) {
		t.Setenv("GITHUB_TOKEN", "gh_abcd")
		url, err := githubPlugin.url("test")
		assert.NoError(t, err)
		actual, err := githubPlugin.request(url)
		assert.NoError(t, err)
		assert.Equal(t, expectedURL, actual.URL.String())
		assert.Equal(t, "token gh_abcd", actual.Header.Get("Authorization"))
	})
}
