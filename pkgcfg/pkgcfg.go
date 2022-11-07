package pkgcfg

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"text/template"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/debug"
	"go.jetpack.io/devbox/nix"
)

const localPkgConfigPath = "DEVBOX_LOCAL_PKG_CONFIG"

type config struct {
	Name            string            `json:"name"`
	Version         string            `json:"version"`
	CreateFiles     map[string]string `json:"create_files"`
	Env             map[string]string `json:"env"`
	localConfigPath string            `json:"-"`
}

func CreateFiles(pkg, rootDir string) error {
	cfg, err := get(pkg, rootDir)
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
		content, err := getFile(cfg, contentPath)
		if err != nil {
			return errors.WithStack(err)
		}
		t, err := template.New(name + "-template").Parse(string(content))
		if err != nil {
			return errors.WithStack(err)
		}
		var buf bytes.Buffer
		if err = t.Execute(&buf, map[string]string{
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
	return nil
}

func Env(pkgs []string, rootDir string) (map[string]string, error) {
	env := map[string]string{}
	for _, pkg := range pkgs {
		cfg, err := get(pkg, rootDir)
		if err != nil {
			return nil, err
		}
		for k, v := range cfg.Env {
			env[k] = v
		}
	}
	return env, nil
}

func get(pkg, rootDir string) (*config, error) {
	if configPath := os.Getenv(localPkgConfigPath); configPath != "" {
		debug.Log("Using local package config at %q", configPath)
		return getLocalConfig(configPath, pkg, rootDir)
	}
	return getConfig(pkg, rootDir)
}

var baseConfigURL = "https://raw.githubusercontent.com/jetpack-io/devbox/main/pkgcfg/package-configuration"

func getConfig(pkg, rootDir string) (*config, error) {
	confURL, err := url.JoinPath(baseConfigURL, pkg+".json")
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := http.Get(confURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return &config{}, nil
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(&config{}, pkg, rootDir, string(content))
}

func getLocalConfig(configPath, pkg, rootDir string) (*config, error) {
	pkgConfigPath := filepath.Join(configPath, pkg+".json")
	if _, err := os.Stat(pkgConfigPath); errors.Is(err, os.ErrNotExist) {
		// We don't need config for all packages and that's fine
		return &config{}, nil
	}
	debug.Log("Reading local package config at %q", pkgConfigPath)
	content, err := os.ReadFile(pkgConfigPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	cfg := &config{localConfigPath: configPath}
	return buildConfig(cfg, pkg, rootDir, string(content))
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
	newname := filepath.Join(root, ".devbox/conf/bin", name)

	// Create bin path just in case it doesn't exist
	if err := os.MkdirAll(filepath.Join(root, ".devbox/conf/bin"), 0755); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Stat(newname); err == nil {
		if err = os.Remove(newname); err != nil {
			return errors.WithStack(err)
		}
	}

	if err := os.Symlink(filePath, newname); err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func getFile(cfg *config, contentPath string) ([]byte, error) {
	if cfg.localConfigPath != "" {
		return os.ReadFile(filepath.Join(cfg.localConfigPath, contentPath))
	}
	confURL, err := url.JoinPath(baseConfigURL, contentPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := http.Get(confURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}
