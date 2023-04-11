package nix

import (
	"crypto/md5"
	"encoding/hex"
	"net/url"
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

func (i *Input) NormalizedName() (string, error) {
	attrPath, err := i.PackageAttributePath()
	if err != nil {
		return "", err
	}
	return i.URLWithoutFragment() + "#" + attrPath, nil
}

// PackageAttributePath returns just the name for non-flakes. For flake
// references is returns the full path to the package in the flake. e.g.
// packages.x86_64-linux.hello
func (i *Input) PackageAttributePath() (string, error) {
	if !i.IsFlake() {
		return i.String(), nil
	}
	infos := search(i.String())
	if len(infos) == 0 {
		return "", usererr.New("Flake \"%s\" was found", i.String())
	} else if len(infos) > 1 {
		return "", usererr.New(
			"Flake \"%s\" is ambiguous. It has multiple packages outputs: %s",
			i.String(),
			lo.Keys(infos),
		)
	}

	return lo.Keys(infos)[0], nil
}

func (i *Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) validateExists() (bool, error) {
	info, err := i.PackageAttributePath()
	return info != "", err
}

func (i *Input) equals(other *Input) bool {
	if i.String() == other.String() {
		return true
	}
	if i.Scheme == other.Scheme &&
		i.Path == other.Path &&
		i.Opaque == other.Opaque &&
		i.normalizedFragment() == other.normalizedFragment() {
		return true
	}

	// check inputs without fragments as optimization. Next step is expensive
	if i.URLWithoutFragment() != other.URLWithoutFragment() {
		return false
	}

	name, err := i.PackageAttributePath()
	if err != nil {
		return false
	}
	otherName, err := other.PackageAttributePath()
	if err != nil {
		return false
	}
	return name == otherName
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
