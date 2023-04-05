package nix

import (
	"crypto/md5"
	"encoding/hex"
	"path/filepath"
	"regexp"
	"strings"
)

type Input string

var pathRegex = regexp.MustCompile(`^path:([^#]*).*$`)
var fragmentRegex = regexp.MustCompile(`^.*#(.*)$`)

// isFlake returns true if the package descriptor has a protocol. For now
// we only support the "path" protocol.
func (i Input) IsFlake() bool {
	return strings.HasPrefix(string(i), "path:")
}

func (i Input) Name() string {
	return filepath.Base(i.path()) + "-" + i.hash()
}

func (i Input) URL(projectDir string) string {
	match := pathRegex.FindStringSubmatch(string(i))
	if len(match) == 0 {
		return ""
	}
	path := match[1]

	if !filepath.IsAbs(path) {
		path = filepath.Join(projectDir, path)
	}

	protocol := strings.Split(string(i), ":")[0]
	return protocol + ":" + path
}

func (i Input) Packages() []string {
	return strings.Split(i.fragment(), ",")
}

func (i Input) fragment() string {
	fragment := fragmentRegex.FindStringSubmatch(string(i))
	if len(fragment) == 0 {
		return ""
	}
	return fragment[1]
}

func (i Input) urlWithFragment(projectDir string) string {
	url := i.URL(projectDir)
	fragment := i.fragment()
	if fragment == "" {
		return url
	}
	return url + "#" + fragment
}

func (i Input) path() string {
	path := pathRegex.FindStringSubmatch(string(i))
	if len(path) == 0 {
		return ""
	}
	return path[1]
}

func (i Input) hash() string {
	hasher := md5.New()
	hasher.Write([]byte(i))
	hash := hasher.Sum(nil)
	shortHash := hex.EncodeToString(hash)[:6]
	return shortHash
}
