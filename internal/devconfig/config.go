package devconfig

import (
	"os"

	"go.jetpack.io/devbox/internal/devbox/shellcmd"
)

// Config represents a base devbox.json as well as any imports it may have.
// TODO: All the functions below will be modified to include all imported configs.
type Config struct {
	Root configFile

	// This will support imports in the future.
	// imported []*Config
}

// Load reads a devbox config file, and validates it.
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	baseConfig, err := loadBytes(b)
	if err != nil {
		return nil, err
	}
	return &Config{Root: *baseConfig}, nil
}

func (c *Config) PackageMutator() *packagesMutator {
	return &c.Root.PackagesMutator
}

func (c *Config) Packages() []Package {
	return c.Root.PackagesMutator.collection
}

// PackagesVersionedNames returns a list of package names with versions.
// NOTE: if the package is unversioned, the version will be omitted (doesn't default to @latest).
//
// example:
// ["package1", "package2@latest", "package3@1.20"]
func (c *Config) PackagesVersionedNames() []string {
	result := make([]string, 0, len(c.Root.PackagesMutator.collection))
	for _, p := range c.Root.PackagesMutator.collection {
		result = append(result, p.VersionedName())
	}
	return result
}

func (c *Config) NixPkgsCommitHash() string {
	return c.Root.NixPkgsCommitHash()
}

func (c *Config) Env() map[string]string {
	return c.Root.Env
}

func (c *Config) InitHook() *shellcmd.Commands {
	return c.Root.InitHook()
}

func (c *Config) Scripts() scripts {
	return c.Root.Scripts()
}

func (c *Config) Hash() (string, error) {
	return c.Root.Hash()
}

func (c *Config) Include() []string {
	return c.Root.Include
}

func (c *Config) IsEnvsecEnabled() bool {
	return c.Root.IsEnvsecEnabled()
}
