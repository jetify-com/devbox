package nix

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

type Input url.URL

func InputFromString(s, projectDir string) *Input {
	u, _ := url.Parse(s)
	if u.Path == "" && u.Opaque != "" && u.Scheme == "path" {
		u.Path = filepath.Join(projectDir, u.Opaque)
		u.Opaque = ""
	}
	return lo.ToPtr(Input(*u))
}

func (i *Input) String() string {
	return (*url.URL)(i).String()
}

// isFlake returns true if the package descriptor has a scheme. For now
// we only support the "path" scheme.
func (i *Input) IsFlake() bool {
	return i.IsLocal() || i.IsGithub()
}

func (i *Input) IsLocal() bool {
	// Technically flakes allows omitting the scheme for local absolute paths, but
	// we don't support that (yet).
	return i.Scheme == "path"
}

func (i *Input) IsGithub() bool {
	return i.Scheme == "github"
}

func (i *Input) Name() string {
	if i.IsLocal() {
		return filepath.Base(i.Path) + "-" + i.hash()
	}
	if i.IsGithub() {
		return "gh-" + strings.Join(strings.Split(i.Opaque, "/"), "-")
	}
	return i.String() + "-" + i.hash()
}

func (i *Input) URLWithoutFragment() string {
	u := *(*url.URL)(i) // get copy
	u.Fragment = ""
	// This will produce urls with extra slashes after the scheme, but that's ok
	return u.String()
}

// Package returns just the name for non-flakes. For flake references is returns
// the full path to the package in the flake.
func (i *Input) Package() string {
	if !i.IsFlake() {
		return i.String()
	}
	p, _ := i.outputPath()
	return p
}

func (i *Input) outputPath() (string, error) {
	infos := search(i.String())
	if len(infos) == 0 {
		return "", ErrPackageNotFound
	}

	system, err := currentSystem()
	if err != nil {
		return "", err
	}

	key := fmt.Sprintf("packages.%s.%s", system, i.normalizedFragment())
	if _, exists := infos[key]; exists {
		return key, nil
	}

	key = fmt.Sprintf("legacyPackages.%s.%s", system, i.normalizedFragment())
	if _, exists := infos[key]; exists {
		return key, nil
	}

	if hasDefault, err := i.hasDefaultPackage(); err != nil {
		return "", err
	} else if hasDefault {
		return "defaultPackage." + system, nil
	}

	return "", usererr.New(
		"Flake \"%s\" was found but package \"%s\" was not found in flake. "+
			"Ensure the flake has a packages output",
		i.Path,
		i.normalizedFragment(),
	)
}

func (i *Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) validateExists() (bool, error) {
	o, err := i.outputPath()
	return o != "", err
}

func (i *Input) equals(o *Input) bool {
	if i.String() == o.String() {
		return true
	}
	return i.Scheme == o.Scheme &&
		i.Path == o.Path &&
		i.Opaque == o.Opaque &&
		i.normalizedFragment() == o.normalizedFragment()
}

// normalizedFragment attempts to return the closest thing to a package name
// from a fragment. A fragment could be:
// * empty string -> default
// * a single package -> package
// * a qualified output (e.g. packages.aarch64-darwin.hello) -> hello
func (i *Input) normalizedFragment() string {
	if i.Fragment == "" {
		return "default"
	}
	parts := strings.Split(i.Fragment, ".")
	return parts[len(parts)-1]
}

var currentSystemCache string

func currentSystem() (string, error) {
	if currentSystemCache == "" {
		cmd := exec.Command(
			"nix", "eval",
			"--impure", "--raw", "--expr",
			"builtins.currentSystem",
		)
		cmd.Args = append(cmd.Args, ExperimentalFlags()...)
		o, err := cmd.Output()
		if err != nil {
			return "", err
		}
		currentSystemCache = strings.TrimSpace(string(o))
	}
	return currentSystemCache, nil
}

type output struct {
	DefaultPackage map[string]map[string]any `json:"defaultPackage"`
}

// hasDefaultPackage returns true if the flake has a defaultPackage output.
// Landau: I'm not sure if this is a standard way of exposing default packages,
// but process-compose does this and we want to support it.
func (i *Input) hasDefaultPackage() (bool, error) {
	cmd := exec.Command(
		"nix", "flake", "show",
		i.URLWithoutFragment(),
		"--json",
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	commandOut, err := cmd.Output()
	if err != nil {
		return false, err
	}
	out := &output{}
	if err = json.Unmarshal(commandOut, out); err != nil {
		return false, err
	}

	if len(out.DefaultPackage) == 0 {
		return false, nil
	}

	system, err := currentSystem()
	if err != nil {
		return false, err
	}

	_, exists := out.DefaultPackage[system]
	return exists, nil
}
