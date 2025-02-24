package pkgtype

import (
	"context"
	"os"
	"strings"

	"go.jetify.com/pkg/runx/impl/registry"
	"go.jetify.com/pkg/runx/impl/runx"
)

const (
	RunXScheme            = "runx"
	RunXPrefix            = RunXScheme + ":"
	githubAPITokenVarName = "DEVBOX_GITHUB_API_TOKEN"
)

var cachedRegistry *registry.Registry

func IsRunX(s string) bool {
	return strings.HasPrefix(s, RunXPrefix)
}

func RunXClient() *runx.RunX {
	return &runx.RunX{
		GithubAPIToken: os.Getenv(githubAPITokenVarName),
	}
}

func RunXRegistry(ctx context.Context) (*registry.Registry, error) {
	if cachedRegistry == nil {
		var err error
		cachedRegistry, err = registry.NewLocalRegistry(ctx, os.Getenv(githubAPITokenVarName))
		if err != nil {
			return nil, err
		}
	}
	return cachedRegistry, nil
}
