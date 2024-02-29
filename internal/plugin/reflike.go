package plugin

import (
	"fmt"
	"path/filepath"
	"strings"

	"go.jetpack.io/devbox/nix/flake"
)

// RefLike is like a flake ref, but in some ways more general. It can be used
// to reference other types of files, e.g. devbox.json.
type RefLike struct {
	flake.Ref
	filename string
}

type Includable interface {
	CanonicalName() string
	Hash() string
	FileContent(subpath string) ([]byte, error)
}

func parseReflike(s string) (Includable, error) {
	ref, err := flake.ParseRef(s)
	if err != nil {
		return nil, err
	}
	reflike := RefLike{ref, pluginConfigName}
	switch ref.Type {
	case flake.TypePath:
		return newLocalPlugin(reflike)
	case flake.TypeGitHub:
		return &githubPlugin{ref: reflike}, nil
	default:
		return nil, fmt.Errorf("unsupported ref type %q", ref.Type)
	}
}

func (r RefLike) withFilename(s string) string {
	if strings.HasSuffix(s, r.filename) {
		return s
	}
	return filepath.Join(s, r.filename)
}
