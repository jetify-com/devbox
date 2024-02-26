package plugin

import (
	"io"
	"net/http"
	"net/url"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/cachehash"
)

type githubPlugin struct {
	RefLike
}

func newGithubPlugin(ref RefLike) (*githubPlugin, error) {
	if ref.Ref.Ref == "" && ref.Rev == "" {
		ref.Ref.Ref = "master"
	}
	return &githubPlugin{RefLike: ref}, nil
}

func (p *githubPlugin) Fetch() ([]byte, error) {
	// Github redirects "master" to "main" in new repos. They don't do the reverse
	// so setting master here is better.
	contentURL, err := url.JoinPath(
		"https://raw.githubusercontent.com/",
		p.Owner,
		p.Repo,
		lo.Ternary(p.Rev == "", "master", p.Rev),
		p.withFilename(p.Dir),
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
			"failed to fetch github import:%s (Status code %d). \nPlease make sure a "+
				"%s file exists in the directory.",
			contentURL,
			res.StatusCode,
			p.filename,
		)
	}
	return io.ReadAll(res.Body)
}

func (p *githubPlugin) CanonicalName() string {
	return p.Owner + "-" + p.Repo
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
		p.Owner,
		p.Repo,
		lo.Ternary(p.Rev == "", "master", p.Rev),
		p.withFilename(p.Dir),
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
			p.String(),
			res.StatusCode,
		)
	}
	return io.ReadAll(res.Body)
}
