package devconfig

import (
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devbox/shellcmd"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/plugin"
)

// Config represents a base devbox.json as well as any included plugins it may have.
type Config struct {
	Root configfile.ConfigFile

	pluginData *plugin.PluginOnlyData // pointer by design, to allow for nil

	included []*Config
}

const defaultInitHook = "echo 'Welcome to devbox!' > /dev/null"

func DefaultConfig() *Config {
	cfg, err := loadBytes([]byte(fmt.Sprintf(`{
  "$schema": "https://raw.githubusercontent.com/jetpack-io/devbox/main/.schema/devbox.schema.json",
  "packages": [],
  "shell": {
    "init_hook": [
      "%s"
    ],
    "scripts": {
      "test": [
        "echo \"Error: no test specified\" && exit 1"
      ]
    }
  }
}
`, defaultInitHook)), "")
	if err != nil {
		panic("default devbox.json is invalid: " + err.Error())
	}
	return cfg
}

// Load reads a devbox config file, and validates it.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return loadBytes(b, filepath.Dir(path))
}

func LoadConfigFromURL(url string) (*Config, error) {
	res, err := http.Get(url)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return loadBytes(data, "")
}

func loadBytes(b []byte, projectDir string) (*Config, error) {
	baseConfig, err := configfile.LoadBytes(b)
	if err != nil {
		return nil, err
	}

	return loadRecursive(baseConfig, projectDir)
}

func loadRecursive(config *configfile.ConfigFile, projectDir string) (*Config, error) {
	included := make([]*Config, 0, len(config.Include))

	for _, importRef := range config.Include {
		pluginConfig, err := plugin.LoadConfigFromInclude(importRef, projectDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}

		includable, err := loadRecursive(&pluginConfig.ConfigFile, projectDir)
		includable.pluginData = &pluginConfig.PluginOnlyData
		if err != nil {
			return nil, errors.WithStack(err)
		}

		included = append(included, includable)
	}

	builtIns, err := plugin.GetBuiltinsForPackages(
		config.PackagesMutator.Collection,
		projectDir,
	)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	for _, builtIn := range builtIns {
		includable, err := loadRecursive(&builtIn.ConfigFile, projectDir)
		if err != nil {
			return nil, errors.WithStack(err)
		}
		pluginData := builtIn.PluginOnlyData
		includable.pluginData = &pluginData
		included = append(included, includable)
	}

	return &Config{
		Root:     *config,
		included: included,
	}, nil
}

func (c *Config) PackageMutator() *configfile.PackagesMutator {
	return &c.Root.PackagesMutator
}

func (c *Config) PluginConfigs() []*plugin.Config {
	configs := []*plugin.Config{}
	for _, i := range c.included {
		configs = append(configs, i.PluginConfigs()...)
	}
	if c.pluginData != nil {
		configs = append(configs, &plugin.Config{
			ConfigFile:     c.Root,
			PluginOnlyData: *c.pluginData,
		})
	}
	return configs
}

func (c *Config) Packages() []configfile.Package {
	packages := []configfile.Package{}
	packagesToRemove := map[string]bool{}

	for _, i := range c.included {
		packages = append(packages, i.Packages()...)
		if i.pluginData.RemoveTriggerPackage {
			if pkg, ok := i.pluginData.Source.(any).(*configfile.Package); ok {
				packagesToRemove[pkg.VersionedName()] = true
			}
		}
	}

	// Packages to remove in built ins only affect the devbox.json where they are defined.
	// They should not remove packages that are part of other imports.
	for _, pkg := range c.Root.PackagesMutator.Collection {
		if !packagesToRemove[pkg.VersionedName()] {
			packages = append(packages, pkg)
		}
	}

	// Keep only the last occurrence of each package (by name).
	return lo.Reverse(lo.UniqBy(
		lo.Reverse(packages),
		func(p configfile.Package) string { return p.Name },
	))
}

// PackagesVersionedNames returns a list of package names with versions.
// NOTE: if the package is unversioned, the version will be omitted (doesn't default to @latest).
//
// example:
// ["package1", "package2@latest", "package3@1.20"]
func (c *Config) PackagesVersionedNames() []string {
	result := make([]string, 0, len(c.Root.PackagesMutator.Collection))
	for _, p := range c.Root.PackagesMutator.Collection {
		result = append(result, p.VersionedName())
	}
	return result
}

func (c *Config) NixPkgsCommitHash() string {
	return c.Root.NixPkgsCommitHash()
}

func (c *Config) Env() map[string]string {
	env := map[string]string{}
	for _, i := range c.included {
		maps.Copy(env, i.Env())
	}
	maps.Copy(env, c.Root.Env)
	return env
}

func (c *Config) InitHook() *shellcmd.Commands {
	commands := shellcmd.Commands{}
	for _, i := range c.included {
		commands.Cmds = append(commands.Cmds, i.InitHook().Cmds...)
	}
	commands.Cmds = append(commands.Cmds, c.Root.InitHook().Cmds...)
	return &commands
}

func (c *Config) Scripts() configfile.Scripts {
	scripts := configfile.Scripts{}
	for _, i := range c.included {
		maps.Copy(scripts, i.Scripts())
	}
	maps.Copy(scripts, c.Root.Scripts())
	return scripts
}

func (c *Config) Hash() (string, error) {
	data := []byte{}
	for _, i := range c.included {
		hash, err := i.Hash()
		if err != nil {
			return "", err
		}
		data = append(data, hash...)
	}
	hash, err := c.Root.Hash()
	if err != nil {
		return "", err
	}
	data = append(data, hash...)
	return cachehash.Bytes(data)
}

func (c *Config) IsEnvsecEnabled() bool {
	for _, i := range c.included {
		if i.IsEnvsecEnabled() {
			return true
		}
	}
	return c.Root.IsEnvsecEnabled()
}
