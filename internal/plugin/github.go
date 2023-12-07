package plugin

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
)

type githubPlugin struct {
	raw      string
	org      string
	repo     string
	revision string
	dir      string
}

// newGithubPlugin returns a plugin that is hosted on github.
// url is of the form org/repo?dir=<dir>
// The (optional) dir must have a plugin.json"
func newGithubPlugin(rawURL string) (*githubPlugin, error) {
	pluginURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}

	parts := strings.SplitN(pluginURL.Path, "/", 3)

	if len(parts) < 2 {
		return nil, usererr.New(
			"invalid github plugin url %q. Must be of the form org/repo/[revision]",
			rawURL,
		)
	}

	plugin := &githubPlugin{
		raw:      rawURL,
		org:      parts[0],
		repo:     parts[1],
		revision: "master",
		dir:      pluginURL.Query().Get("dir"),
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
	h, _ := cachehash.Bytes([]byte(p.CanonicalName()))
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
		p.dir,
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
				"plugin.json file exists in plugin directory.",
			p.raw,
			res.StatusCode,
		)
	}
	return io.ReadAll(res.Body)
}

func (p *githubPlugin) buildConfig(projectDir string) (*config, error) {
	content, err := p.FileContent("plugin.json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(p, projectDir, string(content))
}
