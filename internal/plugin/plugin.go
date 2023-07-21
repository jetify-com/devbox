// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

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
	"go.jetpack.io/devbox/internal/devpkg"

	"go.jetpack.io/devbox/internal/conf"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/impl/shellcmd"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/services"
)

const (
	devboxDirName       = "devbox.d"
	devboxHiddenDirName = ".devbox"
)

var (
	VirtenvPath    = filepath.Join(devboxHiddenDirName, "virtenv")
	VirtenvBinPath = filepath.Join(VirtenvPath, "bin")

	WrapperPath    = filepath.Join(VirtenvPath, ".wrappers")
	WrapperBinPath = filepath.Join(WrapperPath, "bin")
)

type config struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	CreateFiles map[string]string `json:"create_files"`
	Packages    []string          `json:"__packages"`
	Env         map[string]string `json:"env"`
	Readme      string            `json:"readme"`

	Shell struct {
		// InitHook contains commands that will run at shell startup.
		InitHook shellcmd.Commands `json:"init_hook,omitempty"`
	} `json:"shell,omitempty"`
}

func (c *config) ProcessComposeYaml() (string, bool) {
	for file := range c.CreateFiles {
		if strings.HasSuffix(file, "process-compose.yaml") || strings.HasSuffix(file, "process-compose.yml") {
			return file, true
		}
	}
	return "", false
}

func (c *config) Services() (services.Services, error) {
	if file, ok := c.ProcessComposeYaml(); ok {
		return services.FromProcessCompose(file)
	}
	return nil, nil
}

func (m *Manager) Include(included string) error {
	name, err := m.ParseInclude(included)
	if err != nil {
		return err
	}
	err = m.create(name, m.lockfile.Packages[included])
	return err
}

func (m *Manager) Create(pkg *devpkg.Package) error {
	return m.create(pkg, m.lockfile.Packages[pkg.Raw])
}

func (m *Manager) create(pkg Includable, locked *lock.Package) error {
	virtenvPath := filepath.Join(m.ProjectDir(), VirtenvPath)
	cfg, err := getConfigIfAny(pkg, m.ProjectDir())
	if err != nil {
		return err
	}
	if cfg == nil {
		return nil
	}

	name := pkg.CanonicalName()

	// Always create this dir because some plugins depend on it.
	if err = createDir(filepath.Join(virtenvPath, name)); err != nil {
		return err
	}

	debug.Log("Creating files for package %q create files", pkg)
	for filePath, contentPath := range cfg.CreateFiles {
		if !m.shouldCreateFile(locked, filePath) {
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

		if err = m.createFile(pkg, filePath, contentPath, virtenvPath); err != nil {
			return err
		}

	}

	if locked != nil {
		locked.PluginVersion = cfg.Version
	}

	return m.lockfile.Save()
}

func (m *Manager) createFile(
	pkg Includable,
	filePath, contentPath, virtenvPath string,
) error {
	name := pkg.CanonicalName()
	debug.Log("Creating file %q from contentPath: %q", filePath, contentPath)
	content, err := pkg.FileContent(contentPath)
	if err != nil {
		return errors.WithStack(err)
	}
	tmpl, err := template.New(filePath + "-template").Parse(string(content))
	if err != nil {
		return errors.WithStack(err)
	}

	system, err := nix.System()
	if err != nil {
		return err
	}

	var urlForInput, attributePath string

	if pkg, ok := pkg.(*devpkg.Package); ok {
		attributePath, err = pkg.PackageAttributePath()
		if err != nil {
			return err
		}
		urlForInput = pkg.URLForFlakeInput()
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, map[string]any{
		"DevboxDir":            filepath.Join(m.ProjectDir(), devboxDirName, name),
		"DevboxDirRoot":        filepath.Join(m.ProjectDir(), devboxDirName),
		"DevboxProfileDefault": filepath.Join(m.ProjectDir(), nix.ProfilePath),
		"PackageAttributePath": attributePath,
		"Packages":             m.Packages(),
		"System":               system,
		"URLForInput":          urlForInput,
		"Virtenv":              filepath.Join(virtenvPath, name),
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
		if err := createSymlink(m.ProjectDir(), filePath); err != nil {
			return err
		}
	}
	return nil
}

// Env returns the environment variables for the given plugins.
// TODO: We should associate the env variables with the individual plugin
// binaries via wrappers instead of adding to the environment everywhere.
// TODO: this should have PluginManager as receiver so we can build once with
// pkgs, includes, etc
func (m *Manager) Env(
	pkgs []*devpkg.Package,
	includes []string,
	computedEnv map[string]string,
) (map[string]string, error) {
	allPkgs := []Includable{}
	for _, pkg := range pkgs {
		allPkgs = append(allPkgs, pkg)
	}
	for _, included := range includes {
		input, err := m.ParseInclude(included)
		if err != nil {
			return nil, err
		}
		allPkgs = append(allPkgs, input)
	}

	env := map[string]string{}
	for _, pkg := range allPkgs {
		cfg, err := getConfigIfAny(pkg, m.ProjectDir())
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
	return conf.OSExpandEnvMap(env, computedEnv, m.ProjectDir()), nil
}

func buildConfig(pkg Includable, projectDir, content string) (*config, error) {
	cfg := &config{}
	name := pkg.CanonicalName()
	t, err := template.New(name + "-template").Parse(content)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	var buf bytes.Buffer
	if err = t.Execute(&buf, map[string]string{
		"DevboxProjectDir":     projectDir,
		"DevboxDir":            filepath.Join(projectDir, devboxDirName, name),
		"DevboxDirRoot":        filepath.Join(projectDir, devboxDirName),
		"DevboxProfileDefault": filepath.Join(projectDir, nix.ProfilePath),
		"Virtenv":              filepath.Join(projectDir, VirtenvPath, name),
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
	newname := filepath.Join(root, VirtenvBinPath, name)

	// Create bin path just in case it doesn't exist
	if err := os.MkdirAll(filepath.Join(root, VirtenvBinPath), 0755); err != nil {
		return errors.WithStack(err)
	}

	if _, err := os.Lstat(newname); err == nil {
		if err = os.Remove(newname); err != nil {
			return errors.WithStack(err)
		}
	}

	return errors.WithStack(os.Symlink(filePath, newname))
}

func (m *Manager) shouldCreateFile(
	pkg *lock.Package,
	filePath string,
) bool {
	// Only create files in devboxDir if they are not in the lockfile
	pluginInstalled := pkg != nil && pkg.PluginVersion != ""
	if strings.Contains(filePath, devboxDirName) && pluginInstalled {
		return false
	}

	// Hidden .devbox files are always replaceable, so ok to recreate
	if strings.Contains(filePath, devboxHiddenDirName) {
		return true
	}
	_, err := os.Stat(filePath)
	// File doesn't exist, so we should create it.
	return errors.Is(err, fs.ErrNotExist)
}
