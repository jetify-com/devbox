package devconfig

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"slices"

	"github.com/pkg/errors"
	"github.com/tailscale/hujson"
	"go.jetpack.io/devbox/internal/devbox/shellcmd"
	"go.jetpack.io/devbox/nix/flake"
)

// Config represents a base devbox.json as well as any imports it may have.
// TODO: All the functions below will be modified to include all imported configs.
type Config struct {
	Root ConfigFile

	imports []*Config
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
`, defaultInitHook)))
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
	return loadBytes(b)
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
	return loadBytes(data)
}

func loadBytes(b []byte) (*Config, error) {
	jsonb, err := hujson.Standardize(slices.Clone(b))
	if err != nil {
		return nil, err
	}

	ast, err := parseConfig(b)
	if err != nil {
		return nil, err
	}
	baseConfig := &ConfigFile{
		PackagesMutator: packagesMutator{ast: ast},
		ast:             ast,
	}
	if err := json.Unmarshal(jsonb, baseConfig); err != nil {
		return nil, err
	}

	imports := make([]*Config, 0, len(baseConfig.Imports))

	for _, importRef := range baseConfig.Imports {
		ref, _ := flake.ParseRefLike(importRef, "devbox.json")
		childConfig, err := ref.Fetch()
		if err != nil {
			return nil, fmt.Errorf("failed to fetch import %s: %w", importRef, err)
		}
		importConfig, err := loadBytes(childConfig)
		if err != nil {
			return nil, err
		}
		imports = append(imports, importConfig)
	}

	return &Config{
		Root:    *baseConfig,
		imports: imports,
	}, validateConfig(baseConfig)
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
