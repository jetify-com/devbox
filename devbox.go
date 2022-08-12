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
	for _, pkg := range pkgs {
		ok := nix.PkgExists(pkg)
		if !ok {
			return errors.Errorf("Package %s not found.", pkg)
		}
	}
	// TODO: detect duplicates
	d.cfg.Packages = append(d.cfg.Packages, pkgs...)
	return d.saveCfg()
}

func (d *Devbox) Build() error {
	err := d.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	return docker.Build(d.srcDir, docker.BuildOpts{
		Name: "devbox", // TODO: make it configurable.
	})
}

func (d *Devbox) Plan() *planner.BuildPlan {
	basePlan := &planner.BuildPlan{
		Packages: d.cfg.Packages,
	}
	return planner.MergePlans(basePlan, planner.Plan(d.srcDir))
}

// TODO: generate necessary files without modifying src directory.
func (d *Devbox) Generate() error {
	plan := d.Plan()
	return generate(d.srcDir, plan)
}

func (d *Devbox) Shell() error {
	err := d.Generate()
	if err != nil {
		return errors.WithStack(err)
	}
	return nix.Shell(d.srcDir)
}

func (d *Devbox) saveCfg() error {
	cfgPath := filepath.Join(d.srcDir, CONFIG_FILENAME)
	return cuecfg.WriteFile(cfgPath, d.cfg)
}
