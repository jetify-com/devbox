package docgen

import (
	_ "embed"
	"os"
	"text/template"

	"go.jetpack.io/devbox/internal/devbox"
)

//go:embed readme.tmpl
var readmeTemplate string

func GenerateReadme(devbox *devbox.Devbox, path string) error {
	t, err := template.New("readme").Parse(readmeTemplate)
	if err != nil {
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}

	return t.Execute(f, map[string]any{
		"Name":        devbox.Config().Name,
		"Description": devbox.Config().Description,
		"Scripts":     devbox.Config().Scripts(),
		"EnvVars":     devbox.Config().Env,
		"InitHook":    devbox.Config().InitHook(),
		"Packages":    devbox.ConfigPackages(),
	})
}
