package devpkg

import (
	"context"
	"io"
	"net/http"
	"sync"
	"time"

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

// IsInBinaryCache returns true if the package is in the binary cache.
// ALERT: Callers in a perf-sensitive code path should call FillNarInfoCache
// before calling this function.
func (p *Package) IsInBinaryCache() (bool, error) {
	// Patched glibc packages are not in the binary cache.
	if p.PatchGlibc {
		return false, nil
	}
	// Packages with non-default outputs are not to be taken from the binary cache.
	if len(p.Outputs) > 0 {
		return false, nil
	}
	if eligible, err := p.isEligibleForBinaryCache(); err != nil {
		return false, err
	} else if !eligible {
		return false, nil
	}
	return p.fetchNarInfoStatusOnce()
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
		// IMPORTANT: isEligibleForBinaryCache will call resolve() which is NOT
		// concurrency safe. Hence, we call it outside of the go-routine.
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
		pkg := p // copy the loop variable since its used in a closure below
		group.Go(func() error {
			_, err := pkg.fetchNarInfoStatusOnce()
			return err
		})
	}
	return group.Wait()
}

// narInfoStatusFnCache contains cached OnceValues functions that return cache
// status for a package. In the future we can remove this cache by caching
// package objects and ensuring packages are shared globally.
var narInfoStatusFnCache = sync.Map{}

// fetchNarInfoStatusOnce is like fetchNarInfoStatus, but will only ever run
// once and cache the result.
func (p *Package) fetchNarInfoStatusOnce() (bool, error) {
	type inCacheFunc func() (bool, error)
	f, ok := narInfoStatusFnCache.Load(p.Raw)
	if !ok {
		f = inCacheFunc(sync.OnceValues(p.fetchNarInfoStatus))
		f, _ = narInfoStatusFnCache.LoadOrStore(p.Raw, f)
	}
	return f.(inCacheFunc)()
}

// fetchNarInfoStatus fetches the cache status for the package. It returns
// true if cache exists, false otherwise.
// NOTE: This function always performs an HTTP request and should not be called
// more than once per package.
func (p *Package) fetchNarInfoStatus() (bool, error) {
	sysInfo, err := p.sysInfoIfExists()
	if err != nil {
		return false, err
	} else if sysInfo == nil {
		return false, errors.New(
			"sysInfo is nil, but should not be because" +
				" the package is eligible for binary cache",
		)
	}

	pathParts := nix.NewStorePathParts(sysInfo.DefaultStorePath())
	reqURL := BinaryCache + "/" + pathParts.Hash + ".narinfo"
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, reqURL, nil)
	if err != nil {
		return false, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return false, err
	}
	// read the body fully, and close it to ensure the connection is reused.
	_, _ = io.Copy(io.Discard, res.Body)
	defer res.Body.Close()

	return res.StatusCode == 200, nil
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
