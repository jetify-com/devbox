package impl

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/xdg"
)

// we need a more modern commit to get version of process-compose we want
// once the default nixpkgs commit is updated, we can remove this
const nixpkgsUtilityCommit = "f7475ce8950b761d80a13f3f81d2c23fce60c1dd"

// addDevboxUtilityPackage adds a package to the devbox utility profile.
// It's used to install applications devbox might need, like process-compose
// This is an alternative to a global install which would modify a user's
// environment.
func (d *Devbox) addDevboxUtilityPackage(pkg string) error {
	profilePath, err := utilityNixProfilePath()
	if err != nil {
		return err
	}

	utilityRoot, err := utilityDataPath()
	if err != nil {
		return err
	}

	return nix.ProfileInstall(&nix.ProfileInstallArgs{
		NixpkgsCommit: nixpkgsUtilityCommit,
		Package:       pkg,
		ProfilePath:   profilePath,
		ProjectDir:    utilityRoot,
		Writer:        d.writer,
	})
}

func utilityLookPath(binName string) (string, error) {
	binPath, err := utilityBinPath()
	if err != nil {
		return "", err
	}
	absPath := filepath.Join(binPath, binName)
	_, err = os.Stat(absPath)
	if errors.Is(err, fs.ErrNotExist) {
		return "", err
	}
	return absPath, nil
}

func utilityDataPath() (string, error) {
	path := xdg.DataSubpath("devbox/util")
	return path, errors.WithStack(os.MkdirAll(path, 0755))
}

func utilityNixProfilePath() (string, error) {
	path, err := utilityDataPath()
	if err != nil {
		return "", err
	}
	return filepath.Join(path, "profile"), nil
}

func utilityBinPath() (string, error) {
	nixProfilePath, err := utilityNixProfilePath()
	if err != nil {
		return "", err
	}
	return filepath.Join(nixProfilePath, "bin"), nil
}
