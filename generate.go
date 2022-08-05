package devbox

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/axiom/opensource/devbox/cuecfg"
)

//go:embed tmpl
var tmplFS embed.FS

func Generate(path string, cfg *Config) error {
	err := initConfig(path)
	if err != nil {
		return errors.WithStack(err)
	}

	err = writeFromTemplate(path, cfg, "shell.nix")
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func initConfig(path string) error {
	cfgPath := filepath.Join(path, "devbox.json")

	if _, err := os.Stat(cfgPath); err == nil {
		return nil
	}
	return cuecfg.WriteFile(cfgPath, &Config{
		Packages: []string{},
	})
}

func writeFromTemplate(path string, cfg *Config, tmplName string) error {
	tmplPath := fmt.Sprintf("tmpl/%s.tmpl", tmplName)
	t := template.Must(template.New(tmplName+".tmpl").ParseFS(tmplFS, tmplPath))

	f, err := os.Create(filepath.Join(path, tmplName))
	defer func() {
		_ = f.Close()
	}()
	if err != nil {
		return errors.WithStack(err)
	}

	return t.Execute(f, cfg)
}
