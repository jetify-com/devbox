package pkgcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/nix"
)

const confPath = ".devbox/conf"

type config struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Match       string            `json:"match"`
	CreateFiles map[string]string `json:"create_files"`
	Env         map[string]string `json:"env"`
	Readme      string            `json:"readme"`
	Services    Services          `json:"services"`
}

func CreateFilesAndShowReadme(pkg, rootDir string) error {
	cfg, err := getConfig(pkg, rootDir)
	if err != nil {
		return err
	}
	debug.Log("Creating files for package %q create files", pkg)
	for name, contentPath := range cfg.CreateFiles {
		filePath := filepath.Join(rootDir, name)

		dirPath := filepath.Dir(filePath)
		if contentPath == "" {
			dirPath = filePath
		}
		if err = createDir(dirPath); err != nil {
			return errors.WithStack(err)
		}

		if contentPath == "" {
			continue
		}

		debug.Log("Creating file %q", filePath)
		content, err := getFileContent(contentPath)
		if err != nil {
			return errors.WithStack(err)
		}
		t, err := template.New(name + "-template").Parse(string(content))
		if err != nil {
			return errors.WithStack(err)
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, map[string]string{
			"UserRoot":             rootDir,
			"DevboxRoot":           filepath.Join(rootDir, ".devbox"),
			"DevboxProfileDefault": filepath.Join(rootDir, nix.ProfilePath),
		}); err != nil {
			return errors.WithStack(err)
		}
		if err := os.WriteFile(filePath, buf.Bytes(), 0744); err != nil {
			return errors.WithStack(err)
		}
		if err := createSymlink(rootDir, filePath); err != nil {
			return err
		}
	}
	return createEnvFile(pkg, rootDir)

}

func Env(pkgs []string, rootDir string) (map[string]string, error) {
	env := map[string]string{}
	for _, pkg := range pkgs {
		cfg, err := getConfig(pkg, rootDir)
		if err != nil {
			return nil, err
		}
		for k, v := range cfg.Env {
			env[k] = v
		}
	}
	return env, nil
}

func createEnvFile(pkg, rootDir string) error {
	envVars, err := Env([]string{pkg}, rootDir)
	if err != nil {
		return err
	}
	env := ""
	for k, v := range envVars {
		escaped, err := json.Marshal(v)
		if err != nil {
			return errors.WithStack(err)
		}
		env += fmt.Sprintf("export %s=%s\n", k, escaped)
	}
	filePath := filepath.Join(rootDir, confPath, pkg, "/env")
	if err = createDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, []byte(env), 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func buildConfig(cfg *config, pkg, rootDir, content string) (*config, error) {
	t, err := template.New(pkg + "-template").Parse(content)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, map[string]string{
		"DevboxRoot":           filepath.Join(rootDir, ".devbox"),
		"DevboxProfileDefault": filepath.Join(rootDir, nix.ProfilePath),
	}); err != nil {
		return nil, errors.WithStack(err)
	}
	if err = json.Unmarshal(buf.Bytes(), cfg); err != nil {
		return nil, errors.WithStack(err)
	}
	return cfg, nil
}

func createDir(path string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func createSymlink(root, filePath string) error {
	name := filepath.Base(filePath)
	newname := filepath.Join(root, confPath, "bin", name)

	// Create bin path just in case it doesn't exist
	if err := os.MkdirAll(filepath.Join(root, confPath, "/bin"), 0755); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Lstat(newname); err == nil {
		if err = os.Remove(newname); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := os.Symlink(filePath, newname); err != nil {
		return errors.WithStack(err)
	}
	return nil
}
