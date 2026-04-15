package devbox

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.jetify.com/devbox/internal/devpkg"
	"go.jetify.com/devbox/internal/lock"
	"go.jetify.com/devbox/internal/nix"
)

func TestUpdateNewPackageIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, nil)
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{}, // empty
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
}

func TestUpdateNewCurrentSysInfoIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys := currentSystem(t)
	devPkg := devpkg.PackageFromStringWithDefaults(raw, nil)
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys: {
				Outputs: []lock.Output{
					{
						Name:    "out",
						Default: true,
						Path:    "store_path1",
					},
				},
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				// No system infos.
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys)
	require.Equal(t, "store_path1", lockfile.Packages[raw].Systems[sys].Outputs[0].Path)
}

func TestUpdateNewSysInfoIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys1 := currentSystem(t)
	sys2 := "system2"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, nil)
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys1: {
				Outputs: []lock.Output{
					{
						Name:    "out",
						Default: true,
						Path:    "store_path1",
					},
				},
			},
			sys2: {
				Outputs: []lock.Output{
					{
						Name:    "out",
						Default: true,
						Path:    "store_path2",
					},
				},
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Systems: map[string]*lock.SystemInfo{
					sys1: {
						Outputs: []lock.Output{
							{
								Name:    "out",
								Default: true,
								Path:    "store_path1",
							},
						},
					},
					// Missing sys2
				},
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys1)
	require.Contains(t, lockfile.Packages[raw].Systems, sys2)
	require.Equal(t, "store_path2", lockfile.Packages[raw].Systems[sys2].Outputs[0].Path)
}

func TestUpdateOtherSysInfoIsReplaced(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys1 := currentSystem(t)
	sys2 := "system2"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, nil)
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys1: {
				Outputs: []lock.Output{
					{
						Name:    "out",
						Default: true,
						Path:    "store_path1",
					},
				},
			},
			sys2: {
				Outputs: []lock.Output{
					{
						Name:    "out",
						Default: true,
						Path:    "store_path2",
					},
				},
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Systems: map[string]*lock.SystemInfo{
					sys1: {
						Outputs: []lock.Output{
							{
								Name:    "out",
								Default: true,
								Path:    "store_path1",
							},
						},
					},
					sys2: {
						Outputs: []lock.Output{
							{
								Name:    "out",
								Default: true,
								Path:    "mismatching_store_path",
							},
						},
					},
				},
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys1)
	require.Contains(t, lockfile.Packages[raw].Systems, sys2)
	require.Equal(t, "store_path1", lockfile.Packages[raw].Systems[sys1].Outputs[0].Path)
	require.Equal(t, "store_path2", lockfile.Packages[raw].Systems[sys2].Outputs[0].Path)
}

func TestUpdateOnlyTargetedPackagesAreSelected(t *testing.T) {
	d := devboxForTesting(t)

	pkgA := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)
	pkgB := devpkg.PackageFromStringWithDefaults("curl@latest", nil)
	pkgC := devpkg.PackageFromStringWithDefaults("git@latest", nil)

	// Simulate updating only pkgB.
	d.packagesBeingUpdated = []*devpkg.Package{pkgB}

	require.False(t, d.isBeingUpdated(pkgA), "pkgA should not be marked as being updated")
	require.True(t, d.isBeingUpdated(pkgB), "pkgB should be marked as being updated")
	require.False(t, d.isBeingUpdated(pkgC), "pkgC should not be marked as being updated")
}

func TestUpdateAllPackagesSelectedWhenNoneTargeted(t *testing.T) {
	d := devboxForTesting(t)

	pkgA := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)
	pkgB := devpkg.PackageFromStringWithDefaults("curl@latest", nil)

	// Simulate `devbox update` with no args: all packages are in the update list.
	d.packagesBeingUpdated = []*devpkg.Package{pkgA, pkgB}

	require.True(t, d.isBeingUpdated(pkgA))
	require.True(t, d.isBeingUpdated(pkgB))
}

func TestUpdateEmptyUpdateListSelectsNothing(t *testing.T) {
	d := devboxForTesting(t)

	pkg := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)

	// packagesBeingUpdated is nil (default) — no package should be force-refreshed.
	require.False(t, d.isBeingUpdated(pkg))
}

func currentSystem(*testing.T) string {
	sys := nix.System() // NOTE: we could mock this too, if it helps.
	return sys
}
