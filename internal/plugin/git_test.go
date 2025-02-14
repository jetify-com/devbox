package plugin

import (
	"fmt"
	"strings"
	"testing"

	"github.com/samber/lo"
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
			name: "parse github plugin with dir param and ref",
			Include: []flake.Ref{
				{
					Type:  "github",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
					Ref:   "initials/my-branch",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "github",
					Host:  "github.com",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
					Ref:   "initials/my-branch",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/initials/my-branch/mongodb",
		},
		{
			name: "parse github plugin with dir param, rev, and ref",
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
					Host:  "github.com",
					Owner: "jetify-com",
					Repo:  "devbox-plugins",
					Dir:   "mongodb",
					Ref:   "initials/my-branch", // Rev takes precendence over Ref; we exclude the Ref in the URL based on original useage of cmp.Or
					Rev:   "initials",
				},
				name: "jetify-com.devbox-plugins.mongodb",
			},
			expectedURL: "https://raw.githubusercontent.com/jetify-com/devbox-plugins/initials/mongodb",
		},
		{
			name: "parse basic gitlab plugin",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-plugin",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-plugin",
				},
				name: "username.my-plugin",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-plugin/repository/files/raw?ref=main",
		},
		{
			name: "parse gitlab plugin with dir param",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
				},
				name: "username.my-plugin.mongodb",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-plugin/repository/files/mongodb/raw?ref=main",
		},
		{
			name: "parse gitlab plugin with dir param and ref",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Ref:   "some/branch",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Ref:   "some/branch",
				},
				name: "username.my-plugin.mongodb",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-plugin/repository/files/mongodb/raw?ref=some%2Fbranch",
		},
		{
			name: "parse gitlab plugin with dir param and rev",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Rev:   "1234567",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Rev:   "1234567",
				},
				name: "username.my-plugin.mongodb",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-plugin/repository/files/mongodb/raw?ref=1234567",
		},
		{
			name: "parse gitlab plugin with dir param and rev",
			Include: []flake.Ref{
				{
					Type:  "gitlab",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
			},
			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "gitlab",
					Host:  "gitlab.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "mongodb",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
				name: "username.my-plugin.mongodb",
			},
			expectedURL: "https://gitlab.com/api/v4/projects/username%2Fmy-plugin/repository/files/mongodb/raw?ref=1234567",
		},
		{
			name: "parse basic bitbucket plugin",
			Include: []flake.Ref{
				{
					Type:  "bitbucket",
					Owner: "username",
					Repo:  "my-plugin",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "bitbucket",
					Host:  "bitbucket.com",
					Owner: "username",
					Repo:  "my-plugin",
				},
				name: "username.my-plugin",
			},
			expectedURL: "https://api.bitbucket.org/2.0/repositories/username/my-plugin/src/main",
		},
		{
			name: "parse bitbucket plugin with dir param",
			Include: []flake.Ref{
				{
					Type:  "bitbucket",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "bitbucket",
					Host:  "bitbucket.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "https://api.bitbucket.org/2.0/repositories/username/my-plugin/src/main/subdir",
		},
		{
			name: "parse bitbucket plugin with dir param and ref",
			Include: []flake.Ref{
				{
					Type:  "bitbucket",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "bitbucket",
					Host:  "bitbucket.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "https://api.bitbucket.org/2.0/repositories/username/my-plugin/src/some/branch/subdir",
		},
		{
			name: "parse bitbucket plugin with dir param and rev",
			Include: []flake.Ref{
				{
					Type:  "bitbucket",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Rev:   "1234567",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "bitbucket",
					Host:  "bitbucket.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Rev:   "1234567",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "https://api.bitbucket.org/2.0/repositories/username/my-plugin/src/1234567/subdir",
		},
		{
			name: "parse bitbucket plugin with dir param, ref and rev",
			Include: []flake.Ref{
				{
					Type:  "bitbucket",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "bitbucket",
					Host:  "bitbucket.com",
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "https://api.bitbucket.org/2.0/repositories/username/my-plugin/src/1234567/subdir",
		},
		{
			name: "parse basic ssh plugin",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Owner: "username",
					Repo:  "my-plugin",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Owner: "username",
					Repo:  "my-plugin",
				},
				name: "username.my-plugin",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost/username/my-plugin main -o",
		},
		{
			name: "parse ssh plugin with port",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
				},
				name: "username.my-plugin",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost:9999/username/my-plugin main -o",
		},
		{
			name: "parse ssh plugin with port and dir",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost:9999/username/my-plugin main subdir -o",
		},
		{
			name: "parse ssh plugin with port, dir and rev",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Rev:   "1234567",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Rev:   "1234567",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost:9999/username/my-plugin 1234567 subdir -o",
		},
		{
			name: "parse ssh plugin with port, dir and ref",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost:9999/username/my-plugin some/branch subdir -o",
		},
		{
			name: "parse ssh plugin with port, dir, ref and ref",
			Include: []flake.Ref{
				{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
			},

			expected: gitPlugin{
				ref: flake.Ref{
					Type:  "ssh",
					Host:  "localhost",
					Port:  9999,
					Owner: "username",
					Repo:  "my-plugin",
					Dir:   "subdir",
					Ref:   "some/branch",
					Rev:   "1234567",
				},
				name: "username.my-plugin.subdir",
			},
			expectedURL: "git archive --format=tar.gz --remote=ssh://git@localhost:9999/username/my-plugin 1234567 subdir -o",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			actual, err := newGitPluginForTest(testCase.Include[0]) // FIXME: need to evaluate URL
			assert.NoError(t, err)
			assert.Equal(t, &testCase.expected, actual)
			u, err := testCase.expected.url("")
			assert.Nil(t, err)
			assert.Equal(t, testCase.expectedURL, u)
		})
	}
}

func newGitPluginForTest(ref flake.Ref) (*gitPlugin, error) {
	// added because this occurs much earlier in processing within `internal/devconfig/config.go`
	switch ref.Type {
	case flake.TypeGitHub, flake.TypeGitLab, flake.TypeBitBucket:
		ref.Host = fmt.Sprintf("%s.com", ref.Type)
	}

	plugin := &gitPlugin{ref: ref}
	name := strings.ReplaceAll(ref.Dir, "/", "-")
	repoDotted := strings.ReplaceAll(ref.Repo, "/", ".")
	plugin.name = githubNameRegexp.ReplaceAllString(
		strings.Join(lo.Compact([]string{ref.Owner, repoDotted, name}), "."),
		" ",
	)
	return plugin, nil
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
