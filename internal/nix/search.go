package nix

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/debug"
	"go.jetpack.io/devbox/internal/xdg"
	"go.jetpack.io/pkg/filecache"
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
		// TODO implement runx search. Also, move this check outside this function: nix package
		// should not be handling runx logic.
		return map[string]*Info{}, nil
	}
	return searchSystem(url, "" /* system */)
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

	// The `^` is added to indicate we want to show all packages
	cmd := command("search", url, "^" /*regex*/, "--json")
	if system != "" {
		cmd.Args = append(cmd.Args, "--system", system)
	}
	debug.Log("running command: %s\n", cmd)
	out, err := cmd.Output(context.TODO())
	if err != nil {
		// for now, assume all errors are invalid packages.
		// TODO: check the error string for "did not find attribute" and
		// return ErrPackageNotFound only for that case.
		return nil, fmt.Errorf("error searching for pkg %s: %w", url, err)
	}
	return parseSearchResults(out), nil
}

// allowableQuery specifies the regex that queries for SearchNixpkgsAttribute must match.
var allowableQuery = regexp.MustCompile("^github:NixOS/nixpkgs/[0-9a-f]{40}#[^#]+$")

// SearchNixpkgsAttribute is a wrapper around searchSystem that caches results.
// NOTE: we should be very conservative in where we use this function. `nix search`
// accepts generalized `installable regex` as arguments but is slow. For certain
// queries of the form `nixpkgs/<commit-hash>#attribute`, we can know for sure that
// once `nix search` returns a valid result, it will always be the very same result.
// Hence we can cache it locally and answer future queries fast, by not calling `nix search`.
func SearchNixpkgsAttribute(query string) (map[string]*Info, error) {
	if !allowableQuery.MatchString(query) {
		return nil, errors.Errorf("invalid query: %s, must match regex: %s", query, allowableQuery)
	}

	key := cacheKey(query)

	// Check if the query was already cached, and return the result if so
	cache := filecache.New(
		"devbox/nix",
		filecache.WithCacheDir[map[string]*Info](xdg.CacheSubpath("")),
	)

	if results, err := cache.Get(key); err == nil {
		return results, nil
	} else if !filecache.IsCacheMiss(err) {
		return nil, err // genuine error
	}

	// If not cached, or an update is needed, then call searchSystem
	infos, err := searchSystem(query, "" /*system*/)
	if err != nil {
		return nil, err
	}

	// Save the results to the cache
	// TODO savil: add a SetForever API that does not expire. Time based expiration is not needed here
	// because we're caching results that are guaranteed to be stable.
	// TODO savil: Make filecache.cache a public struct so it can be passed into other functions
	const oneYear = 12 * 30 * 24 * time.Hour
	if err := cache.Set(key, infos, oneYear); err != nil {
		return nil, err
	}

	return infos, nil
}

// cacheKey sanitizes the search query to be a valid unix filename.
// This cache key is used as the filename to store the cache value, and having a
// representation of the query is important for debuggability.
func cacheKey(query string) string {
	// Replace disallowed characters with underscores.
	re := regexp.MustCompile(`[:/#@+]`)
	sanitized := re.ReplaceAllString(query, "_")

	// Remove any remaining invalid characters.
	sanitized = regexp.MustCompile(`[^\w\.-]`).ReplaceAllString(sanitized, "")

	// Ensure the filename doesn't exceed the maximum length.
	const maxLen = 255
	if len(sanitized) > maxLen {
		sanitized = sanitized[:maxLen]
	}

	return sanitized
}
