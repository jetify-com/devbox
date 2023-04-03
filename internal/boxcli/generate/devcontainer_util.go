package generate

import (
	"embed"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"go.jetpack.io/devbox/internal/debug"
)

type devcontainerObject struct {
	Name           string          `json:"name"`
	Build          *build          `json:"build"`
	Customizations *customizations `json:"customizations"`
	RemoteUser     string          `json:"remoteUser"`
}

type build struct {
	Dockerfile string `json:"dockerfile"`
	Context    string `json:"context"`
}

type customizations struct {
	Vscode *vscode `json:"vscode"`
}

type vscode struct {
	Settings   any      `json:"settings"`
	Extensions []string `json:"extensions"`
}

// Creates a Dockerfile in path and writes devcontainerDockerfile.tmpl's content into it
func CreateDockerfile(tmplFS embed.FS, path string) error {
	// create dockerfile
	file, err := os.Create(filepath.Join(path, "Dockerfile"))
	if err != nil {
		return err
	}
	defer file.Close()
	// get dockerfile content
	tmplName := "devcontainerDockerfile.tmpl"
	t := template.Must(template.ParseFS(tmplFS, "tmpl/"+tmplName))
	// write content into file
	return t.Execute(file, nil)
}

// CreateDevcontainer creates a devcontainer.json in path and writes getDevcontainerContent's output into it
func CreateDevcontainer(path string, pkgs []string) error {
	// create devcontainer.json file
	file, err := os.Create(filepath.Join(path, "devcontainer.json"))
	if err != nil {
		return err
	}
	defer file.Close()
	// get devcontainer.json's content
	devcontainerContent := getDevcontainerContent(pkgs)
	devcontainerFileBytes, err := json.MarshalIndent(devcontainerContent, "", "  ")
	if err != nil {
		return err
	}
	// writing devcontainer's content into json file
	_, err = file.Write(devcontainerFileBytes)
	return err
}

func CreateEnvrc(tmplFS embed.FS, path string) error {
	// create .envrc file
	file, err := os.Create(filepath.Join(path, ".envrc"))
	if err != nil {
		return err
	}
	defer file.Close()
	// get .envrc content
	tmplName := "envrc.tmpl"
	t := template.Must(template.ParseFS(tmplFS, "tmpl/"+tmplName))
	// write content into file
	return t.Execute(file, nil)
}

func getDevcontainerContent(pkgs []string) *devcontainerObject {
	// object that gets written in devcontainer.json
	devcontainerContent := &devcontainerObject{
		// For format details, see https://aka.ms/devcontainer.json. For config options, see the README at:
		// https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/debian
		Name: "Devbox Remote Container",
		Build: &build{
			Dockerfile: "./Dockerfile",
			Context:    "..",
		},
		Customizations: &customizations{
			Vscode: &vscode{
				Settings: map[string]any{
					// Add custom vscode settings for remote environment here
				},
				Extensions: []string{
					"jetpack-io.devbox",
					// Add custom vscode extensions for remote environment here
				},
			},
		},
		// Comment out to connect as root instead. More info: https://aka.ms/vscode-remote/containers/non-root.
		RemoteUser: "devbox",
	}

	// match only python3 or python3xx as package names
	py3pattern, err := regexp.Compile(`(python3)$|(python3[0-9]{1,2})$`)
	if err != nil {
		debug.Log("Failed to compile regex")
		return nil
	}
	for _, pkg := range pkgs {
		if py3pattern.MatchString(pkg) {
			// Setup python3 interpreter path to devbox in the container
			devcontainerContent.Customizations.Vscode.Settings = map[string]any{
				"python.defaultInterpreterPath": "/code/.devbox/nix/profile/default/bin/python3",
			}
			// add python extension if a python3 package is installed
			devcontainerContent.Customizations.Vscode.Extensions =
				append(devcontainerContent.Customizations.Vscode.Extensions, "ms-python.python")
		}
		if strings.Contains(pkg, "go_1_") || pkg == "go" {
			devcontainerContent.Customizations.Vscode.Extensions =
				append(devcontainerContent.Customizations.Vscode.Extensions, "golang.go")
		}
		// TODO: add support for other common languages
	}
	return devcontainerContent
}
