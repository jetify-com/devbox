package devconfig

import (
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/build"
	"go.jetpack.io/devbox/internal/cachehash"
	"go.jetpack.io/devbox/internal/devbox/shellcmd"
	"go.jetpack.io/devbox/internal/devconfig/configfile"
	"go.jetpack.io/devbox/internal/lock"
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
		"$schema": "https://raw.githubusercontent.com/jetify-com/devbox/%s/.schema/devbox.schema.json",
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
	`,
		lo.Ternary(build.IsDev, "main", build.Version),
		defaultInitHook,
	)))
	if err != nil {
		panic("default devbox.json is invalid: " + err.Error())
	}
	return cfg
}

func IsDefault(path string) bool {
	cfg, err := readFromFile(path)
	if err != nil {
		return false
	}
	return cfg.Root.Equals(&DefaultConfig().Root)
}

func LoadForTest(path string) (*Config, error) {
	return readFromFile(path)
}

func readFromFile(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	config, err := loadBytes(b)
	if err != nil {
		return nil, err
	}
	config.Root.AbsRootPath = path
	return config, nil
}

func LoadConfigFromURL(ctx context.Context, url string) (*Config, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return loadBytes(data)
}

func loadBytes(b []byte) (*Config, error) {
	root, err := configfile.LoadBytes(b)
	if err != nil {
		return nil, err
	}

	return &Config{
		Root: *root,
	}, nil
}

func (c *Config) LoadRecursive(lockfile *lock.File) error {
	return c.loadRecursive(lockfile, map[string]bool{}, "" /*cyclePath*/)
}

// loadRecursive loads all the included plugins and their included plugins, etc.
// seen should be a cloned map because loading plugins twice is allowed if they
// are in different paths.
func (c *Config) loadRecursive(
	lockfile *lock.File,
	seen map[string]bool,
	cyclePath string,
) error {
	included := make([]*Config, 0, len(c.Root.Include))

	for _, includeRef := range c.Root.Include {
		pluginConfig, err := plugin.LoadConfigFromInclude(
			includeRef, lockfile, filepath.Dir(c.Root.AbsRootPath))
		if err != nil {
			return errors.WithStack(err)
		}

		newCyclePath := fmt.Sprintf("%s -> %s", cyclePath, includeRef)
		if seen[pluginConfig.Source.Hash()] {
			// Note that duplicate includes are allowed if they are in different paths
			// e.g. 2 different plugins can include the same plugin.
			// We do not allow a single plugin to include duplicates.
			return errors.Errorf(
				"circular or duplicate include detected:\n%s", newCyclePath)
		}
		seen[pluginConfig.Source.Hash()] = true

		includable := createIncludableFromPluginConfig(pluginConfig)

		if err := includable.loadRecursive(
			lockfile, maps.Clone(seen), newCyclePath); err != nil {
			return errors.WithStack(err)
		}

		included = append(included, includable)
	}

	builtIns, err := plugin.GetBuiltinsForPackages(
		c.Root.TopLevelPackages(),
		lockfile,
	)
	if err != nil {
		return errors.WithStack(err)
	}

	for _, builtIn := range builtIns {
		includable := &Config{
			Root:       builtIn.ConfigFile,
			pluginData: &builtIn.PluginOnlyData,
		}
		newCyclePath := fmt.Sprintf("%s -> %s", cyclePath, builtIn.Source.LockfileKey())
		if err := includable.loadRecursive(
			lockfile, maps.Clone(seen), newCyclePath); err != nil {
			return errors.WithStack(err)
		}
		included = append(included, includable)
	}

	c.included = included
	return nil
}

func (c *Config) PackageMutator() *configfile.PackagesMutator {
	return &c.Root.PackagesMutator
}

func (c *Config) IncludedPluginConfigs() []*plugin.Config {
	configs := []*plugin.Config{}
	for _, i := range c.included {
		configs = append(configs, i.IncludedPluginConfigs()...)
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
			packagesToRemove[i.pluginData.Source.LockfileKey()] = true
		}
	}

	// Packages to remove in built ins only affect the devbox.json where they are defined.
	// They should not remove packages that are part of other imports.
	for _, pkg := range c.Root.TopLevelPackages() {
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
	result := make([]string, 0, len(c.Root.TopLevelPackages()))
	for _, p := range c.Root.TopLevelPackages() {
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
	return cachehash.Bytes(data), nil
}

func (c *Config) IsEnvsecEnabled() bool {
	for _, i := range c.included {
		if i.IsEnvsecEnabled() {
			return true
		}
	}
	return c.Root.IsEnvsecEnabled()
}

func createIncludableFromPluginConfig(pluginConfig *plugin.Config) *Config {
	includable := &Config{
		Root:       pluginConfig.ConfigFile,
		pluginData: &pluginConfig.PluginOnlyData,
	}
	if localPlugin, ok := pluginConfig.Source.(*plugin.LocalPlugin); ok {
		includable.Root.AbsRootPath = localPlugin.Path()
	}
	return includable
}
