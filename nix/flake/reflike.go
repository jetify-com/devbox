package flake

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

// RefLike is like a flake ref, but in some ways more general. It can be used
// to reference other types of files, e.g. devbox.json.
type RefLike struct {
	Ref
	filename string
}

func ParseRefLike(s, filename string) (RefLike, error) {
	r, e := ParseRef(s)
	return RefLike{r, filename}, e
}

func (r RefLike) Fetch() ([]byte, error) {
	switch r.Type {
	case TypePath:
		return os.ReadFile(r.withFilename(r.Path))
	case TypeGitHub:
		return r.fetchGithub()
	default:
		return nil, fmt.Errorf("unsupported ref type %q", r.Type)
	}
}

// TODO: This is almost copy paste of plugins/github.go. We should refactor this
func (r RefLike) fetchGithub() ([]byte, error) {
	// Github redirects "master" to "main" in new repos. They don't do the reverse
	// so setting master here is better.
	contentURL, err := url.JoinPath(
		"https://raw.githubusercontent.com/",
		r.Owner,
		r.Repo,
		lo.Ternary(r.Rev == "", "master", r.Rev),
		r.withFilename(r.Dir),
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
			r.filename,
		)
	}
	return io.ReadAll(res.Body)
}

func (r RefLike) withFilename(s string) string {
	if strings.HasSuffix(s, r.filename) {
		return s
	}
	return filepath.Join(s, r.filename)
}
