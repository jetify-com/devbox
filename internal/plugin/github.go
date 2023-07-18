package plugin

import (
	"io"
	"net/http"
	"net/url"
	"strings"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cuecfg"
)

type githubPlugin struct {
	org      string
	repo     string
	revision string
}

// newGithubPlugin returns a plugin that is hosted on github.
// url is of the form org/repo
// The repo must have a devbox.json file in the root of the repo.
func newGithubPlugin(url string) (*githubPlugin, error) {
	parts := strings.Split(url, "/")

	if len(parts) < 2 || len(parts) > 3 {
		return nil, usererr.New(
			"invalid github plugin url %q. Must be of the form org/repo/[revision]",
			url,
		)
	}

	p := &githubPlugin{
		org:      parts[0],
		repo:     parts[1],
		revision: "master",
	}

	if len(parts) == 3 {
		p.revision = parts[2]
	}

	return p, nil
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
		return nil,
			usererr.New("failed to get %s. Status code %d", contentURL, res.StatusCode)
	}
	return io.ReadAll(res.Body)
}
