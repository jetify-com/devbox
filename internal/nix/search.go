package nix

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/xdg"
)

var (
	ErrPackageNotFound     = errors.New("package not found")
	ErrPackageNotInstalled = errors.New("package not installed")
)

type Info struct {
	// attribute key is different in flakes vs legacy so we should only use it
	// if we know exactly which version we are using
	AttributeKey string `json:"attribute"`
	PName        string `json:"pname"`
	Summary      string `json:"summary"`
	Version      string `json:"version"`
}

func (i *Info) String() string {
	return fmt.Sprintf("%s-%s", i.PName, i.Version)
}

func Search(url string) (map[string]*Info, error) {
	if strings.HasPrefix(url, "runx:") {
		// TODO implement runx search
		return map[string]*Info{}, nil
	}
	return searchSystemUsingCache(url, "")
}

func parseSearchResults(data []byte) map[string]*Info {
	var results map[string]map[string]any
	err := json.Unmarshal(data, &results)
	if err != nil {
		panic(err)
	}
	infos := map[string]*Info{}
	for key, result := range results {
		infos[key] = &Info{
			AttributeKey: key,
			PName:        result["pname"].(string),
			Version:      result["version"].(string),
		}
	}
	return infos
}

// PkgExistsForAnySystem is a bit slow (~600ms). Only use it if there's already
// been an error and we want to provide a better error message.
func PkgExistsForAnySystem(pkg string) bool {
	systems := []string{
		// Check most common systems first.
		"x86_64-linux",
		"x86_64-darwin",
		"aarch64-linux",
		"aarch64-darwin",

		"armv5tel-linux",
		"armv6l-linux",
		"armv7l-linux",
		"i686-linux",
		"mipsel-linux",
		"powerpc64le-linux",
		"riscv64-linux",
	}
	for _, system := range systems {
		results, _ := searchSystem(pkg, system)
		if len(results) > 0 {
			return true
		}
	}
	return false
}

func searchSystem(url, system string) (map[string]*Info, error) {
	// Eventually we may pass a writer here, but for now it is safe to use stderr
	writer := os.Stderr
	// Search will download nixpkgs if it's not already downloaded. Adding this
	// check here provides a slightly better UX.
	if IsGithubNixpkgsURL(url) {
		hash := HashFromNixPkgsURL(url)
		// purposely ignore error here. The function already prints an error.
		// We don't want to panic or stop execution if we can't prefetch.
		_ = EnsureNixpkgsPrefetched(writer, hash)
	}

	cmd := exec.Command("nix", "search", "--json", url)
	cmd.Args = append(cmd.Args, ExperimentalFlags()...)
	if system != "" {
		cmd.Args = append(cmd.Args, "--system", system)
	}
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output()
	if err != nil {
		// for now, assume all errors are invalid packages.
		return nil, fmt.Errorf("error searching for pkg %s: %w", url, err)
	}
	return parseSearchResults(out), nil
}

type searchSystemCache struct {
	QueryToInfo map[string]map[string]*Info `json:"query_to_info"`
}

const (
	searchSystemCacheSubDir   = "devbox/nix"
	searchSystemCacheFileName = "search-system-cache.json"
)

var cache = searchSystemCache{}

func searchSystemUsingCache(url, system string) (map[string]*Info, error) {
	if system != "" {
		return searchSystem(url, system)
	}

	if cache.QueryToInfo == nil {
		contents, err := readSearchSystemCacheFile()
		if err != nil {
			return nil, err
		}
		cache.QueryToInfo = contents
	}

	if result := cache.QueryToInfo[url]; result != nil {
		return result, nil
	}

	info, err := searchSystem(url, system)
	if err != nil {
		return nil, err
	}

	cache.QueryToInfo[url] = info
	if err := writeSearchSystemCacheFile(cache.QueryToInfo); err != nil {
		return nil, err
	}

	return info, nil
}

func readSearchSystemCacheFile() (map[string]map[string]*Info, error) {
	contents, err := os.ReadFile(xdg.CacheSubpath(filepath.Join(searchSystemCacheSubDir, searchSystemCacheFileName)))
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]map[string]*Info), nil
		}
		return nil, err
	}
	var result map[string]map[string]*Info
	if err := json.Unmarshal(contents, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func writeSearchSystemCacheFile(contents map[string]map[string]*Info) error {
	buf := bytes.Buffer{}
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	err := enc.Encode(contents)
	if err != nil {
		return err
	}
	dir := xdg.CacheSubpath(searchSystemCacheSubDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	path := filepath.Join(dir, searchSystemCacheFileName)
	return os.WriteFile(path, buf.Bytes(), 0o644)
}
