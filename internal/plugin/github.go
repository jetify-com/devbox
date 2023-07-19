package plugin

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
)

type githubPlugin struct {
	raw      string
	org      string
	repo     string
	revision string
	fragment string
}

// newGithubPlugin returns a plugin that is hosted on github.
// url is of the form org/repo#name
// The repo must have a [name].json in the root of the repo. If fragment is
// not set, it defaults to "default"
func newGithubPlugin(url string) (*githubPlugin, error) {
	path, fragment, _ := strings.Cut(url, "#")

	parts := strings.Split(path, "/")

	if len(parts) < 2 || len(parts) > 3 {
		return nil, usererr.New(
			"invalid github plugin url %q. Must be of the form org/repo/[revision]",
			url,
		)
	}

	plugin := &githubPlugin{
		raw:      url,
		org:      parts[0],
		repo:     parts[1],
		revision: "master",
		fragment: fragment,
	}

	if len(parts) == 3 {
		plugin.revision = parts[2]
	}

	return plugin, nil
}

func (p *githubPlugin) CanonicalName() string {
	return p.org + "-" + p.repo
}

func (p *githubPlugin) Hash() string {
	h, _ := cuecfg.Hash(p.CanonicalName())
	return h
}

func (p *githubPlugin) FileContent(subpath string) ([]byte, error) {
	// Github redirects "master" to "main" in new repos. They don't do the reverse
	// so setting master here is better.
	contentURL, err := url.JoinPath(
		"https://raw.githubusercontent.com/",
		p.org,
		p.repo,
		p.revision,
		subpath,
	)
	if err != nil {
		return nil, err
	}

	res, err := http.Get(contentURL)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return nil, usererr.New(
			"failed to get plugin github:%s (Status code %d). \nPlease make sure a "+
				"[name].json or default.json file exists in the root of the repo.",
			p.raw,
			res.StatusCode,
		)
	}
	return io.ReadAll(res.Body)
}

func (p *githubPlugin) buildConfig(projectDir string) (*config, error) {
	configName, _ := lo.Coalesce(p.fragment, "default")
	content, err := p.FileContent(configName + ".json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(p, projectDir, string(content))
}
