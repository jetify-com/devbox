package nix

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
	"go.jetpack.io/devbox/internal/lockfile"
	"go.jetpack.io/devbox/internal/searcher"
)

type Input struct {
	url.URL
	lockfile *lockfile.Lockfile
}

func InputFromString(s string, l *lockfile.Lockfile) *Input {
	u, _ := url.Parse(s)
	if u.Path == "" && u.Opaque != "" && u.Scheme == "path" {
		u.Path = filepath.Join(l.ProjectDir(), u.Opaque)
		u.Opaque = ""
	}
	return &Input{*u, l}
}

// IsFlake returns true if the package descriptor has a scheme. For now
// we only support the "path" scheme.
func (i *Input) IsFlake() bool {
	return i.IsLocal() || i.IsGithub() || i.IsDevboxPackage()
}

func (i *Input) IsLocal() bool {
	// Technically flakes allows omitting the scheme for local absolute paths, but
	// we don't support that (yet).
	return i.Scheme == "path"
}

func (i *Input) IsDevboxPackage() bool {
	if !featureflag.VersionedPackages.Enabled() {
		return false
	}
	if i.Scheme != "" {
		return false
	}
	return searcher.Client().IsVersionedPackage(i.String())
}

func (i *Input) IsGithub() bool {
	return i.Scheme == "github"
}

var inputNameRegex = regexp.MustCompile("[^a-zA-Z0-9-]+")

func (i *Input) Name() string {
	result := ""
	if i.IsLocal() {
		result = filepath.Base(i.Path) + "-" + i.hash()
	} else if i.IsGithub() {
		result = "gh-" + strings.Join(strings.Split(i.Opaque, "/"), "-")
	} else {
		result = i.String() + "-" + i.hash()
	}
	return inputNameRegex.ReplaceAllString(result, "-")
}

func (i *Input) URLForInput() string {
	if i.IsDevboxPackage() {
		resolved, err := i.lockfile.Resolve(i.String())
		if err != nil {
			panic(err)
			// TODO(landau): handle error
		}
		withoutFragment, _, _ := strings.Cut(resolved, "#")
		return withoutFragment
	}
	return i.urlWithoutFragment()
}

func (i *Input) URLForInstall() (string, error) {
	if i.IsDevboxPackage() {
		return i.lockfile.Resolve(i.String())
	}
	attrPath, err := i.PackageAttributePath()
	if err != nil {
		return "", err
	}
	return i.urlWithoutFragment() + "#" + attrPath, nil
}

// PackageAttributePath returns just the name for non-flakes. For flake
// references is returns the full path to the package in the flake. e.g.
// packages.x86_64-linux.hello
func (i *Input) PackageAttributePath() (string, error) {
	if !i.IsFlake() {
		return i.String(), nil
	}

	var infos map[string]*Info
	if i.IsDevboxPackage() {
		path, err := i.lockfile.Resolve(i.String())
		if err != nil {
			return "", err
		}
		infos = search(path)
	} else {
		infos = search(i.String())
	}

	if len(infos) == 1 {
		return lo.Keys(infos)[0], nil
	}

	// If ambiguous, try to find a default output
	if len(infos) > 1 && i.Fragment == "" {
		for key := range infos {
			if strings.HasSuffix(key, ".default") {
				return key, nil
			}
		}
		for key := range infos {
			if strings.HasPrefix(key, "defaultPackage.") {
				return key, nil
			}
		}
	}

	// Still ambiguous, return error
	if len(infos) > 1 {
		outputs := fmt.Sprintf("It has %d possible outputs", len(infos))
		if len(infos) < 10 {
			outputs = "It has the following possible outputs: \n" +
				strings.Join(lo.Keys(infos), ", ")
		}
		return "", usererr.New(
			"Flake \"%s\" is ambiguous. %s",
			i.String(),
			outputs,
		)
	}

	return "", usererr.New("Flake \"%s\" was not found", i.String())
}

func (i *Input) urlWithoutFragment() string {
	u := i.URL // get copy
	u.Fragment = ""
	// This will produce urls with extra slashes after the scheme, but that's ok
	return u.String()
}

func (i *Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i.String()))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}

func (i *Input) validateExists() (bool, error) {
	if i.IsDevboxPackage() {
		return searcher.Exists(i.canonicalName(), i.version())
	}
	info, err := i.PackageAttributePath()
	return info != "", err
}

func (i *Input) equals(other *Input) bool {
	if i.String() == other.String() {
		return true
	}

	// check inputs without fragments as optimization. Next step is expensive
	if i.URLForInput() != other.URLForInput() {
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

// canonicalName returns the name of the package without the version
// it only applies to devbox packages
func (i *Input) canonicalName() string {
	if !i.IsDevboxPackage() {
		return ""
	}
	name, _, _ := strings.Cut(i.Path, "@")
	return name
}

// version returns the version of the package
// it only applies to devbox packages
func (i *Input) version() string {
	if !i.IsDevboxPackage() {
		return ""
	}
	_, version, _ := strings.Cut(i.Path, "@")
	return version
}
