package devpkg

import (
	"context"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/boxcli/featureflag"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
	"go.jetpack.io/devbox/internal/vercheck"
	"golang.org/x/sync/errgroup"
)

// BinaryCache is the store from which to fetch this package's binaries.
// It is used as FromStore in builtins.fetchClosure.
const BinaryCache = "https://cache.nixos.org"

// isNarInfoInCache checks if the .narinfo for this package is in the `BinaryCache`.
// This cannot be a field on the Package struct, because that struct
// is constructed multiple times in a request (TODO: we could fix that).
var isNarInfoInCache = struct {
	// The key is the `Package.Raw` string.
	status map[string]bool
	lock   sync.RWMutex
	// re-use httpClient to re-use the connection
	httpClient http.Client
}{
	status:     map[string]bool{},
	httpClient: http.Client{},
}

// IsInBinaryCache returns true if the package is in the binary cache.
// ALERT: Callers in a perf-sensitive code path should call FillNarInfoCache
// before calling this function.
func (p *Package) IsInBinaryCache() (bool, error) {

	if eligible, err := p.isEligibleForBinaryCache(); err != nil {
		return false, err
	} else if !eligible {
		return false, nil
	}

	// Check if the narinfo is present in the binary cache
	isNarInfoInCache.lock.RLock()
	status, statusExists := isNarInfoInCache.status[p.Raw]
	isNarInfoInCache.lock.RUnlock()
	if !statusExists {
		// Fallback to synchronously filling the nar info cache
		if err := p.fillNarInfoCache(); err != nil {
			return false, err
		}

		// Check again
		isNarInfoInCache.lock.RLock()
		status, statusExists = isNarInfoInCache.status[p.Raw]
		isNarInfoInCache.lock.RUnlock()
		if !statusExists {
			return false, errors.Errorf(
				"narInfo cache miss: %v. Should be filled by now",
				p.Raw,
			)
		}
	}
	return status, nil
}

// FillNarInfoCache checks the remote binary cache for the narinfo of each
// package in the list, and caches the result.
// Callers of IsInBinaryCache may call this function first as a perf-optimization.
func FillNarInfoCache(ctx context.Context, packages ...*Package) error {
	if !featureflag.RemoveNixpkgs.Enabled() {
		return nil
	}

	eligiblePackages := []*Package{}
	for _, p := range packages {
		// NOTE: isEligibleForBinaryCache also ensures the package is
		// resolved in the lockfile, which must be done before the concurrent
		// section in this function below.
		isEligible, err := p.isEligibleForBinaryCache()
		// If the package is not eligible or there is an error in determining that, then skip it.
		if isEligible && err == nil {
			eligiblePackages = append(eligiblePackages, p)
		}
	}
	if len(eligiblePackages) == 0 {
		return nil
	}

	// Pre-compute values read in fillNarInfoCache
	// so they can be read from multiple go-routines without locks
	_, err := nix.Version()
	if err != nil {
		return err
	}
	_ = nix.System()

	group, _ := errgroup.WithContext(ctx)
	for _, p := range eligiblePackages {
		// If the package's NarInfo status is already known, skip it
		isNarInfoInCache.lock.RLock()
		_, ok := isNarInfoInCache.status[p.Raw]
		isNarInfoInCache.lock.RUnlock()
		if ok {
			continue
		}
		pkg := p // copy the loop variable since its used in a closure below
		group.Go(func() error {
			err := pkg.fillNarInfoCache()
			if err != nil {
				// default to false if there was an error, so we don't re-try
				isNarInfoInCache.lock.Lock()
				isNarInfoInCache.status[pkg.Raw] = false
				isNarInfoInCache.lock.Unlock()
			}
			return err
		})
	}
	return group.Wait()
}

// fillNarInfoCache fills the cache value for the narinfo of this package,
// assuming it is eligible for the binary cache. Callers are responsible
// for checking isEligibleForBinaryCache before calling this function.
//
// NOTE: this must be concurrency safe.
func (p *Package) fillNarInfoCache() error {
	sysInfo, err := p.sysInfoIfExists()
	if err != nil {
		return err
	} else if sysInfo == nil {
		return errors.New(
			"sysInfo is nil, but should not be because" +
				" the package is eligible for binary cache",
		)
	}

	pathParts := newStorePathParts(sysInfo.StorePath)
	reqURL := BinaryCache + "/" + pathParts.hash + ".narinfo"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, reqURL, nil)
	if err != nil {
		return err
	}
	res, err := isNarInfoInCache.httpClient.Do(req)
	if err != nil {
		return err
	}
	// read the body fully, and close it to ensure the connection is reused.
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	isNarInfoInCache.lock.Lock()
	isNarInfoInCache.status[p.Raw] = res.StatusCode == 200
	isNarInfoInCache.lock.Unlock()
	return nil
}

// isEligibleForBinaryCache returns true if we have additional metadata about
// the package to query it from the binary cache.
func (p *Package) isEligibleForBinaryCache() (bool, error) {
	sysInfo, err := p.sysInfoIfExists()
	if err != nil {
		return false, err
	}
	return sysInfo != nil, nil
}

// sysInfoIfExists returns the system info for the user's system. If the sysInfo
// is missing, then nil is returned
// NOTE: this is called from multiple go-routines and needs to be concurrency safe.
// Hence, we compute nix.Version, nix.System and lockfile.Resolve prior to calling this
// function from within a goroutine.
func (p *Package) sysInfoIfExists() (*lock.SystemInfo, error) {
	if !featureflag.RemoveNixpkgs.Enabled() {
		return nil, nil
	}

	if !p.isVersioned() {
		return nil, nil
	}

	version, err := nix.Version()
	if err != nil {
		return nil, err
	}

	// enable for nix >= 2.17
	if vercheck.SemverCompare(version, "2.17.0") < 0 {
		return nil, err
	}

	entry, err := p.lockfile.Resolve(p.Raw)
	if err != nil {
		return nil, err
	}

	userSystem := nix.System()

	if entry.Systems == nil {
		return nil, nil
	}

	// Check if the user's system's info is present in the lockfile
	sysInfo, ok := entry.Systems[userSystem]
	if !ok {
		return nil, nil
	}
	return sysInfo, nil
}

// storePath are the constituent parts of
// /nix/store/<hash>-<name>-<version>
//
// This is a helper struct for analyzing the string representation
type storePathParts struct {
	hash    string
	name    string
	version string
}

// newStorePathParts splits a Nix store path into its hash, name and version
// components in the same way that Nix does.
//
// See https://nixos.org/manual/nix/stable/language/builtins.html#builtins-parseDrvName
func newStorePathParts(path string) storePathParts {
	path = strings.TrimPrefix(path, "/nix/store/")
	// path is now <hash>-<name>-<version

	hash, name := path[:32], path[33:]
	dashIndex := 0
	for i, r := range name {
		if dashIndex != 0 && !unicode.IsLetter(r) {
			return storePathParts{hash: hash, name: name[:dashIndex], version: name[i:]}
		}
		dashIndex = 0
		if r == '-' {
			dashIndex = i
		}
	}
	return storePathParts{hash: hash, name: name}
}
