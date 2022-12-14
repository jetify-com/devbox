package pkgcfg

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/nix"
)

const (
	VirtenvBinPath = ".devbox/virtenv/bin"
	VirtenvPath    = ".devbox/virtenv"
)

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
	for filePath, contentPath := range cfg.CreateFiles {

		if fileAlreadyExistsAndNotReplaceable(filePath) {
			continue
		}

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
		t, err := template.New(filePath + "-template").Parse(string(content))
		if err != nil {
			return errors.WithStack(err)
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, map[string]string{
			"DevboxDir":            filepath.Join(rootDir, "devbox.d", pkg),
			"DevboxDirRoot":        filepath.Join(rootDir, "devbox.d"),
			"DevboxProfileDefault": filepath.Join(rootDir, nix.ProfilePath),
			"Virtenv":              filepath.Join(rootDir, ".devbox", "virtenv", pkg),
		}); err != nil {
			return errors.WithStack(err)
		}
		var fileMode fs.FileMode = 0644
		if strings.Contains(filePath, "bin/") {
			fileMode = 0755
		}

		if err := os.WriteFile(filePath, buf.Bytes(), fileMode); err != nil {
			return errors.WithStack(err)
		}
		if fileMode == 0755 {
			if err := createSymlink(rootDir, filePath); err != nil {
				return err
			}
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
	filePath := filepath.Join(rootDir, VirtenvPath, pkg, "/env")
	if err = createDir(filepath.Dir(filePath)); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, []byte(env), 0644); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func buildConfig(pkg, rootDir, content string) (*config, error) {
	cfg := &config{}
	t, err := template.New(pkg + "-template").Parse(content)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, map[string]string{
		"DevboxDir":            filepath.Join(rootDir, "devbox.d", pkg),
		"DevboxDirRoot":        filepath.Join(rootDir, "devbox.d"),
		"DevboxProfileDefault": filepath.Join(rootDir, nix.ProfilePath),
		"Virtenv":              filepath.Join(rootDir, ".devbox", "virtenv", pkg),
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
	newname := filepath.Join(root, VirtenvPath, "bin", name)

	// Create bin path just in case it doesn't exist
	if err := os.MkdirAll(filepath.Join(root, VirtenvPath, "/bin"), 0755); err != nil {
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

func fileAlreadyExistsAndNotReplaceable(filePath string) bool {
	// Hidden .devbox files are always replaceable
	if strings.HasPrefix(filePath, ".devbox") {
		return false
	}
	_, err := os.Stat(filePath)
	// File doesn't exist, so we should create it
	return err == nil
}
