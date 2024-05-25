package plugin

import (
	"errors"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/nix/flake"
)

type gitlabPlugin struct {
	ref  flake.Ref
	name string
}

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

func (p *gitlabPlugin) FileContent(subpath string) ([]byte, error) {
	return []byte(subpath), nil
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
