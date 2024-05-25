package plugin

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/nix/flake"
	"go.jetpack.io/pkg/filecache"
)

type gitlabPlugin struct {
	ref  flake.Ref
	name string
}

var gitlabCache = filecache.New[[]byte]("devbox/plugin/gitlab")

func newGitlabPlugin(ref flake.Ref) (*gitlabPlugin, error) {
	plugin := &gitlabPlugin{ref: ref}
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

func (p *gitlabPlugin) CanonicalName() string {
	return p.name
}

func (p *gitlabPlugin) request(contentURL string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, contentURL, nil)
	if err != nil {
		return nil, err
	}

	// Add github token to request if available
	glToken := os.Getenv("GITLAB_TOKEN") // TODO: @GITLAB_PLUGIN Is this right?

	if glToken != "" {
		authValue := fmt.Sprintf("token %s", glToken)
		req.Header.Add("Authorization", authValue)
	}

	return req, nil
}

func (p *gitlabPlugin) FileContent(subpath string) ([]byte, error) {
	contentURL, err := p.url(subpath)

	if err != nil {
		return nil, err
	}

	return gitlabCache.GetOrSet(
		contentURL,
		func() ([]byte, time.Duration, error) {
			req, err := p.request(contentURL)

			if err != nil {
				return nil, 0, err
			}

			client := &http.Client{}
			res, err := client.Do(req)

			if err != nil {
				debug.Log(err.Error())
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

			debug.Log(string(body))

			if err != nil {
				return nil, 0, err
			}
			// Cache for 24 hours. Once we store the plugin in the lockfile, we
			// should cache this indefinitely and only invalidate if the plugin
			// is updated.
			return body, 24 * time.Hour, nil
		},
	)
}

func (p *gitlabPlugin) url(subpath string) (string, error) {
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

func (p *gitlabPlugin) Hash() string {
	return cachehash.Bytes([]byte(p.ref.String()))
}

func (p *gitlabPlugin) LockfileKey() string {
	return p.ref.String()
}

func (p *gitlabPlugin) Fetch() ([]byte, error) {
	content, err := p.FileContent(pluginConfigName)
	if err != nil {
		return nil, err
	}
	return jsonPurifyPluginContent(content)
}
