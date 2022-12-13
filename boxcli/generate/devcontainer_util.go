package generate

import (
	"embed"
	"encoding/json"
	"html/template"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
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
		return errors.WithStack(err)
	}
	// get dockerfile content
	tmplName := "devcontainerDockerfile.tmpl"
	t := template.Must(template.ParseFS(tmplFS, "tmpl/"+tmplName))
	// write content into file
	err = t.Execute(file, nil)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

// Creates a devcontainer.json in path and writes getDevcontainerContent's output into it
func CreateDevcontainer(path string, pkgs []string) error {

	// create devcontainer.json file
	file, err := os.Create(filepath.Join(path, "devcontainer.json"))
	if err != nil {
		return errors.WithStack(err)
	}
	// get devcontainer.json's content
	devcontainerContent := getDevcontainerContent(pkgs)
	devcontainerFileBytes, err := json.MarshalIndent(devcontainerContent, "", "  ")
	if err != nil {
		return errors.WithStack(err)
	}
	// writing devcontainer's content into json file
	_, err = file.Write(devcontainerFileBytes)
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
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

	for _, pkg := range pkgs {
		if strings.Contains(pkg, "python3") {
			devcontainerContent.Customizations.Vscode.Settings = map[string]any{
				"python.defaultInterpreterPath": "/devbox/.devbox/nix/profile/default/bin/python3",
			}
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
