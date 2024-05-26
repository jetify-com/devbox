package plugin

import (
	"encoding/json"
	"fmt"
	"regexp"

	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/nix/flake"
)

type Includable interface {
	CanonicalName() string
	FileContent(subpath string) ([]byte, error)
	Hash() string
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
	case flake.TypeGit:
		fallthrough
	case flake.TypeBitBucket:
		fallthrough
	case flake.TypeGitLab:
		fallthrough
	case flake.TypeGitHub:
		return newGitPlugin(ref)
	default:
		return nil, fmt.Errorf("unsupported ref type %q", ref.Type)
	}
}

type fetcher interface {
	Includable
	Fetch() ([]byte, error)
}

var (
	nameRegex      = regexp.MustCompile(`^[a-zA-Z0-9_\- ]+$`)
	errNameMissing = usererr.New("'name' is missing")
)

func getPluginNameFromContent(plugin fetcher) (string, error) {
	content, err := plugin.Fetch()
	if err != nil {
		return "", err
	}
	m := map[string]any{}
	if err := json.Unmarshal(content, &m); err != nil {
		return "", err
	}
	name, ok := m["name"].(string)
	if !ok || name == "" {
		return "",
			fmt.Errorf("%w in plugin %s", errNameMissing, plugin.LockfileKey())
	}
	if !nameRegex.MatchString(name) {
		return "", usererr.New(
			"plugin %s has an invalid name %q. Name must match %s",
			plugin.LockfileKey(), name, nameRegex,
		)
	}
	return name, nil
}
