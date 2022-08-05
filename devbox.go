package devbox

import (
	"fmt"
	"os"

	"github.com/pkg/errors"
)

type Devbox struct {
	workdir string
}

func Open(path string) (*Devbox, error) {
	// If we can't access the directory (either because it doesn not exist or
	// because we don't have permissions), return an error.
	if _, err := os.Stat(path); err != nil {
		return nil, errors.WithStack(err)
	}

	box := &Devbox{
		workdir: path,
	}
	return box, nil
}

func (d *Devbox) Init() error {
	fmt.Printf("TODO: devbox init %s\n", d.workdir)
	return nil
}

func (d *Devbox) Add(pkgs ...string) error {
	fmt.Printf("TODO: devbox add %s\n", pkgs)
	return nil
}

func (d *Devbox) Build() error {
	fmt.Printf("TODO: devbox build %s\n", d.workdir)
	return nil
}

func (d *Devbox) Generate() error {
	fmt.Printf("TODO: devbox build %s\n", d.workdir)
	return nil
}

func (d *Devbox) Shell() error {
	fmt.Printf("TODO: devbox shell %s\n", d.workdir)
	return nil
}
