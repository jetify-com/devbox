package nix

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
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
	if u.Path == "" && u.Opaque != "" {
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
	// Technically flakes allows omitting the scheme for absolute paths, but
	// we don't support that (yet).
	return i.Scheme == "path"
}

func (i *Input) Name() string {
	return filepath.Base(i.Path) + "-" + i.hash()
}

func (i *Input) URLWithoutFragment() string {
	u := *(*url.URL)(i) // get copy
	u.Fragment = ""
	// This will produce urls with extra slashes after the scheme, but that's ok
	return u.String()
}

func (i *Input) Packages() []string {
	if !i.IsFlake() {
		return []string{i.String()}
	}
	if i.Fragment == "" {
		return []string{"default"}
	}
	return strings.Split(i.Fragment, ",")
}

func (i *Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) validateExists() (bool, error) {
	system, err := currentSystem()
	if err != nil {
		return false, err
	}

	outputs, err := outputs(i.Path)
	if err != nil {
		return false, err
	}

	fragment := i.Fragment
	if fragment == "" {
		fragment = "default" // if no package is specified, check for default.
	}

	if _, exists := outputs.Packages[system][fragment]; exists {
		return true, nil
	}

	if _, exists := outputs.LegacyPackages[system][fragment]; exists {
		return true, nil
	}

	// Another way to specify a default package is to output it as
	// defaultPackage.${system}
	if _, exists := outputs.DefaultPackage[system]; exists && i.Fragment == "" {
		return true, nil
	}

	return false, usererr.New(
		"Flake \"%s\" was found but package \"%s\" was not found in flake. "+
			"Ensure the flake has a packages output",
		i.Path,
		fragment,
	)
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

func currentSystem() (string, error) {
	cmd := exec.Command(
		"nix", "eval",
		"--impure", "--raw", "--expr",
		"builtins.currentSystem",
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	o, err := cmd.Output()
	return string(o), err
}

type output struct {
	LegacyPackages map[string]map[string]any `json:"legacyPackages"`
	Packages       map[string]map[string]any `json:"packages"`
	DefaultPackage map[string]map[string]any `json:"defaultPackage"`
}

func outputs(path string) (*output, error) {
	cmd := exec.Command(
		"nix", "flake", "show",
		path,
		"--json", "--legacy",
	)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	commandOut, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	out := &output{}
	return out, json.Unmarshal(commandOut, out)
}
