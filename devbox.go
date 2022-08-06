package devbox

import (
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/axiom/opensource/devbox/cuecfg"
	"go.jetpack.io/axiom/opensource/devbox/docker"
	"go.jetpack.io/axiom/opensource/devbox/nix"
	"go.jetpack.io/axiom/opensource/devbox/planner"
)

type Devbox struct {
	cfg    *Config
	srcDir string
}

const CONFIG_FILENAME = "devbox.json"

func Init(dir string) (bool, error) {
	cfgPath := filepath.Join(dir, CONFIG_FILENAME)
	return cuecfg.InitFile(cfgPath, &Config{})
}

func Open(dir string) (*Devbox, error) {
	cfgPath := filepath.Join(dir, CONFIG_FILENAME)

	cfg, err := ReadConfig(cfgPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	box := &Devbox{
		cfg:    cfg,
		srcDir: dir,
	}
	return box, nil
}

func (d *Devbox) Add(pkgs ...string) error {
	// TODO: validate packages and detect duplicates.
	d.cfg.Packages = append(d.cfg.Packages, pkgs...)
	return d.saveCfg()
}

func (d *Devbox) Build() error {
	return docker.Build(d.srcDir)
}

func (d *Devbox) Plan() *planner.BuildPlan {
	// TODO: should the BuildPlan struct type be part of 'devbox' instead of 'planner'
	return planner.Plan(d.srcDir)
}

// TODO: for now 'generate' is a manual step, but it should happen
// automatically instead (and the files should be hidden).
func (d *Devbox) Generate() error {
	return generate(d.srcDir, d.cfg)
}

func (d *Devbox) Shell() error {
	return nix.Shell(d.srcDir)
}

func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, CONFIG_FILENAME)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}
