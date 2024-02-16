package docgen

import (
	_ "embed"
	"os"
	"text/template"

	"go.jetpack.io/devbox/internal/devbox"
	"go.jetpack.io/devbox/internal/fileutil"
)

//go:embed readme.tmpl
var defaultReadmeTemplate string

const (
	defaultName         = "README.md"
	defaultTemplateName = "readme.tmpl"
)

func GenerateReadme(
	devbox *devbox.Devbox,
	outputPath, templatePath string,
) error {
	readmeTemplate := defaultReadmeTemplate
	if templatePath != "" {
		readmeTemplateBytes, err := os.ReadFile(templatePath)
		if err != nil {
			return err
		}
		readmeTemplate = string(readmeTemplateBytes)
	} else if fileutil.Exists(defaultTemplateName) {
		readmeTemplateBytes, err := os.ReadFile(defaultTemplateName)
		if err != nil {
			return err
		}
		readmeTemplate = string(readmeTemplateBytes)
	}

	tmpl, err := template.New("readme").Parse(readmeTemplate)
	if err != nil {
		return err
	}

	if outputPath == "" {
		outputPath = defaultName
	}

	f, err := os.Create(outputPath)
	if err != nil {
		return err
	}

	return tmpl.Execute(f, map[string]any{
		"Name":        devbox.Config().Root.Name,
		"Description": devbox.Config().Root.Description,
		"Scripts":     devbox.Config().Scripts(),
		"EnvVars":     devbox.Config().Env(),
		"InitHook":    devbox.Config().InitHook(),
		"Packages":    devbox.ConfigPackages(),
	})
}

func SaveDefaultReadmeTemplate(outputPath string) error {
	if outputPath == "" {
		outputPath = defaultTemplateName
	}
	return os.WriteFile(outputPath, []byte(defaultReadmeTemplate), 0o644)
}
