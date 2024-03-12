package plugin

import (
	"fmt"

	"go.jetpack.io/devbox/nix/flake"
)

type Includable interface {
	CanonicalName() string
	Hash() string
	FileContent(subpath string) ([]byte, error)
	LockfileKey() string
}

func parseIncludable(includableRef, workingDir string) (Includable, error) {
	ref, err := flake.ParseRef(includableRef)
	if err != nil {
		return nil, err
	}
	switch ref.Type {
	case flake.TypePath:
		return newLocalPlugin(ref, workingDir)
	case flake.TypeGitHub:
		return &githubPlugin{ref: ref}, nil
	default:
		return nil, fmt.Errorf("unsupported ref type %q", ref.Type)
	}
}
