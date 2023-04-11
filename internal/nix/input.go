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
	"go.jetpack.io/devbox/internal/boxcli/usererr"
)

type Input struct {
	url.URL
}

func InputFromString(s, projectDir string) *Input {
	u, _ := url.Parse(s)
	if u.Path == "" && u.Opaque != "" && u.Scheme == "path" {
		u.Path = filepath.Join(projectDir, u.Opaque)
		u.Opaque = ""
	}
	return &Input{*u}
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

func (i *Input) URLWithoutFragment() string {
	u := i.URL // get copy
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
