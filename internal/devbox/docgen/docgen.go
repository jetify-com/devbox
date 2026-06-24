package docgen

import (
	_ "embed"
	"os"
	"text/template"

	"go.jetify.com/devbox/internal/devbox"
	"go.jetify.com/devbox/internal/fileutil"
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

	cfg := devbox.Config()
	scripts := cfg.Scripts().
		WithRelativePaths(devbox.ProjectDir()).
		InOrder(cfg.ScriptOrder())

	return tmpl.Execute(f, map[string]any{
		"Name":        cfg.Root.Name,
		"Description": cfg.Root.Description,
		"Scripts":     scripts,
		"EnvVars":     cfg.Env(),
		"InitHook":    cfg.InitHook(),
		"Packages":    devbox.TopLevelPackages(),
		// TODO add includes
	})
}

func SaveDefaultReadmeTemplate(outputPath string) error {
	if outputPath == "" {
		outputPath = defaultTemplateName
	}
	return os.WriteFile(outputPath, []byte(defaultReadmeTemplate), 0o644)
}
