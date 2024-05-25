package plugin

import (
	"cmp"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/pkg/filecache"
)

var githubCache = filecache.New[[]byte]("devbox/plugin/github")
var gitlabCache = filecache.New[[]byte]("devbox/plugin/gitlab")

type gitPlugin struct {
	ref  flake.Ref
	name string
}

// Github only allows alphanumeric, hyphen, underscore, and period in repo names.
// but we clean up just in case.
var githubNameRegexp = regexp.MustCompile("[^a-zA-Z0-9-_.]+")

func newGitPlugin(ref flake.Ref) (*gitPlugin, error) {
	plugin := &gitPlugin{ref: ref}
	// For backward compatibility, we don't strictly require name to be present
	// in github plugins. If it's missing, we just use the directory as the name.
	name, err := getPluginNameFromContent(plugin)
	if err != nil && !errors.Is(err, errNameMissing) {
		return nil, err
	}
	if name == "" {
		name = strings.ReplaceAll(ref.Dir, "/", "-")
	}
	plugin.name = githubNameRegexp.ReplaceAllString(
		strings.Join(lo.Compact([]string{ref.Owner, ref.Repo, name}), "."),
		" ",
	)
	return plugin, nil
}

func (p *gitPlugin) Fetch() ([]byte, error) {
	content, err := p.FileContent(pluginConfigName)
	if err != nil {
		return nil, err
	}
	return jsonPurifyPluginContent(content)
}

func (p *gitPlugin) CanonicalName() string {
	return p.name
}

func (p *gitPlugin) Hash() string {
	return cachehash.Bytes([]byte(p.ref.String()))
}

func (p *gitPlugin) FileContent(subpath string) ([]byte, error) {
	contentURL, err := p.url(subpath)
	debug.Log(contentURL)

	if err != nil {
		return nil, err
	}

	callable := func() ([]byte, time.Duration, error) {
		req, err := p.request(contentURL)
		if err != nil {
			return nil, 0, err
		}

		client := &http.Client{}
		res, err := client.Do(req)
		if err != nil {
			return nil, 0, err
		}
		defer res.Body.Close()
		if res.StatusCode != http.StatusOK {
			return nil, 0, usererr.New(
				"failed to get plugin %s @ %s (Status code %d). \nPlease make "+
					"sure a plugin.json file exists in plugin directory.",
				p.LockfileKey(),
				req.URL.String(),
				res.StatusCode,
			)
		}
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return nil, 0, err
		}
		// Cache for 24 hours. Once we store the plugin in the lockfile, we
		// should cache this indefinitely and only invalidate if the plugin
		// is updated.
		return body, 24 * time.Hour, nil
	}

	switch p.ref.Type {

	case flake.TypeGitHub:
		return githubCache.GetOrSet(contentURL, callable)
	case flake.TypeGitLab:
		return gitlabCache.GetOrSet(contentURL, callable)
	case flake.TypeBitBucket:
		fallthrough // TODO
	default:
		return nil, err
	}

}

func (p *gitPlugin) url(subpath string) (string, error) {
	debug.Log(p.ref.Type)
	switch p.ref.Type {
	case flake.TypeGitLab:
		return p.gitlabUrl(subpath)
	case flake.TypeGitHub:
		return p.githubUrl(subpath)
	case flake.TypeBitBucket:
		fallthrough // TODO
	default:
		return "", nil
	}
}

func (p *gitPlugin) githubUrl(subpath string) (string, error) {
	// Github redirects "master" to "main" in new repos. They don't do the reverse
	// so setting master here is better.
	return url.JoinPath(
		"https://raw.githubusercontent.com/",
		p.ref.Owner,
		p.ref.Repo,
		cmp.Or(p.ref.Rev, p.ref.Ref, "master"),
		p.ref.Dir,
		subpath,
	)
}

func (p *gitPlugin) gitlabUrl(subpath string) (string, error) {
	project, err := url.JoinPath(p.ref.Owner, p.ref.Repo)

	if err != nil {
		return "", err
	}

	file, err := url.JoinPath(p.ref.Dir, subpath)

	if err != nil {
		return "", err
	}

	path, err := url.JoinPath(
		"https://gitlab.com/api/v4/projects",
		url.PathEscape(project),
		"repository",
		"files",
		url.PathEscape(file),
		"raw",
	)

	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(path)

	if err != nil {
		return "", err
	}

	query := parsed.Query()
	query.Add("ref", cmp.Or(p.ref.Rev, p.ref.Ref, "main"))
	parsed.RawQuery = query.Encode()

	return parsed.String(), nil
}

func (p *gitPlugin) request(contentURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, contentURL, nil)
	if err != nil {
		return nil, err
	}

	// Add github token to request if available
	ghToken := os.Getenv("GITHUB_TOKEN")

	if ghToken != "" {
		authValue := fmt.Sprintf("token %s", ghToken)
		req.Header.Add("Authorization", authValue)
	}

	return req, nil
}

func (p *gitPlugin) LockfileKey() string {
	return p.ref.String()
}
