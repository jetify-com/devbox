package docgen

import (
	_ "embed"
	"maps"
	"os"
	"strings"
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

	services, err := devbox.Services()
	if err != nil {
		return err
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
		"EnvVars":     envWithRelativePaths(cfg.Env(), devbox.ProjectDir()),
		"InitHook":    cfg.InitHook(),
		"Packages":    devbox.TopLevelPackages(),
		"Services":    services,
		// TODO add includes
	})
}

// envWithRelativePaths returns a copy of env where occurrences of the absolute
// project directory are replaced with ".". This keeps generated READMEs free of
// machine-specific absolute paths (such as a user's home directory) that plugins
// expand into environment variables like PGDATA and PGHOST. It mirrors the
// behavior of configfile.Scripts.WithRelativePaths, which is already applied to
// scripts in the generated README.
func envWithRelativePaths(env map[string]string, projectDir string) map[string]string {
	if projectDir == "" {
		return maps.Clone(env)
	}
	result := make(map[string]string, len(env))
	for key, value := range env {
		result[key] = strings.ReplaceAll(value, projectDir, ".")
	}
	return result
}

func SaveDefaultReadmeTemplate(outputPath string) error {
	if outputPath == "" {
		outputPath = defaultTemplateName
	}
	return os.WriteFile(outputPath, []byte(defaultReadmeTemplate), 0o644)
}
