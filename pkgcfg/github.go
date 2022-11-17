package pkgcfg

import (
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

var baseConfigURL = "https://raw.githubusercontent.com/jetpack-io/devbox/main/pkgcfg/package-configuration"

const githubDirContentAPI = "https://api.github.com/repos/jetpack-io/devbox/contents/pkgcfg/package-configuration"

func getConfig(pkg, rootDir string) (*config, error) {
	confURL, err := getBestConfigPath(pkg)
	if err != nil {
		return nil, errors.WithStack(err)
	} else if confURL == "" {
		return &config{}, nil
	}
	resp, err := http.Get(confURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return buildConfig(&config{}, pkg, rootDir, string(content))
}

func getFile(cfg *config, contentPath string) ([]byte, error) {
	if cfg.localConfigPath != "" {
		return os.ReadFile(filepath.Join(cfg.localConfigPath, contentPath))
	}
	confURL, err := url.JoinPath(baseConfigURL, contentPath)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	resp, err := http.Get(confURL)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func getBestConfigPath(pkg string) (string, error) {
	resp, err := http.Get(githubDirContentAPI)
	if err != nil {
		return "", errors.WithStack(err)
	}
	defer resp.Body.Close()
	var files []struct {
		Name string `json:"name"`
	}
	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := json.Unmarshal(content, &files); err != nil {
		return "", errors.WithStack(err)
	}

	// Try to find perfect match first
	for _, file := range files {
		if file.Name == pkg+".json" {
			return url.JoinPath(baseConfigURL, file.Name)
		}
	}
	for _, file := range files {
		if wildcardMatch(file.Name, pkg+".json") {
			return url.JoinPath(baseConfigURL, file.Name)
		}
	}
	return "", nil
}

func wildcardMatch(filename, pkg string) bool {
	re := regexp.MustCompile(`^` + strings.ReplaceAll(filename, "*", ".*") + `$`)
	return re.MatchString(pkg)
}
