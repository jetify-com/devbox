package plugin

import (
	"bytes"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/conf"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl/shellcmd"
	"go.jetpack.io/devbox/internal/nix"
)

const (
	devboxDirName       = "devbox.d"
	devboxHiddenDirName = ".devbox"
	VirtenvPath         = ".devbox/virtenv"
)

var WrapperPath = filepath.Join(VirtenvPath, ".wrappers")
var WrapperBinPath = filepath.Join(WrapperPath, "bin")

type config struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Match       string            `json:"match"`
	CreateFiles map[string]string `json:"create_files"`
	Env         map[string]string `json:"env"`
	Readme      string            `json:"readme"`
	Services    Services          `json:"services"`

	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook shellcmd.Commands `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`
}

func (m *Manager) CreateFilesAndShowReadme(pkg, projectDir string) error {
	cfg, err := getConfigIfAny(pkg, projectDir)
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}

	// Always create this dir because some plugins depend on it.
	if err = createDir(filepath.Join(projectDir, VirtenvPath, pkg)); err != nil {
		return err
	}

	debug.Log("Creating files for package %q create files", pkg)
	for filePath, contentPath := range cfg.CreateFiles {
		if !m.shouldCreateFile(filePath) {
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
			"DevboxConfigDir":      projectDir,
			"DevboxDir":            filepath.Join(projectDir, devboxDirName, pkg),
			"DevboxDirRoot":        filepath.Join(projectDir, devboxDirName),
			"DevboxProfileDefault": filepath.Join(projectDir, nix.ProfilePath),
			"Virtenv":              filepath.Join(projectDir, devboxHiddenDirName, "virtenv", pkg),
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
			if err := createSymlink(projectDir, filePath); err != nil {
				return err
			}
		}
	}
	return nil
}

// Env returns the environment variables for the given plugins.
// TODO: We should associate the env variables with the individual plugin
// binaries via wrappers instead of adding to the environment everywhere.
func Env(
	pkgs []string,
	projectDir string,
	computedEnv map[string]string,
) (map[string]string, error) {
	env := map[string]string{}
	for _, pkg := range pkgs {
		cfg, err := getConfigIfAny(pkg, projectDir)
		if err != nil {
			return nil, err
		}
		if cfg == nil {
			continue
		}
		for k, v := range cfg.Env {
			env[k] = v
		}
	}
	return conf.OSExpandEnvMap(env, projectDir, computedEnv), nil
}

func buildConfig(pkg, projectDir, content string) (*config, error) {
	cfg := &config{}
	t, err := template.New(pkg + "-template").Parse(content)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, map[string]string{
		"DevboxProjectDir":     projectDir,
		"DevboxDir":            filepath.Join(projectDir, devboxDirName, pkg),
		"DevboxDirRoot":        filepath.Join(projectDir, devboxDirName),
		"DevboxProfileDefault": filepath.Join(projectDir, nix.ProfilePath),
		"Virtenv":              filepath.Join(projectDir, devboxHiddenDirName, "virtenv", pkg),
	}); err != nil {
		return nil, errors.WithStack(err)
	}

	return cfg, errors.WithStack(json.Unmarshal(buf.Bytes(), cfg))
}

func createDir(path string) error {
	if path == "" {
		return nil
	}
	return errors.WithStack(os.MkdirAll(path, 0755))
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

	return errors.WithStack(os.Symlink(filePath, newname))
}

func (m *Manager) shouldCreateFile(filePath string) bool {
	// Only create devboxDir files in add mode.
	if strings.Contains(filePath, devboxDirName) && !m.addMode {
		return false
	}

	// Hidden .devbox files are always replaceable, so ok to recreate
	if strings.Contains(filePath, devboxHiddenDirName) {
		return true
	}
	_, err := os.Stat(filePath)
	// File doesn't exist, so we should create it.
	return os.IsNotExist(err)
}
