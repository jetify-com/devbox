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
	devbox := devboxForTesting(t)

	pkgA := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)
	pkgB := devpkg.PackageFromStringWithDefaults("curl@latest", nil)
	pkgC := devpkg.PackageFromStringWithDefaults("git@latest", nil)

	// Simulate updating only pkgB.
	devbox.packagesBeingUpdated = []*devpkg.Package{pkgB}

	require.False(t, devbox.isBeingUpdated(pkgA), "pkgA should not be marked as being updated")
	require.True(t, devbox.isBeingUpdated(pkgB), "pkgB should be marked as being updated")
	require.False(t, devbox.isBeingUpdated(pkgC), "pkgC should not be marked as being updated")
}

func TestUpdateAllPackagesSelectedWhenNoneTargeted(t *testing.T) {
	devbox := devboxForTesting(t)

	pkgA := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)
	pkgB := devpkg.PackageFromStringWithDefaults("curl@latest", nil)

	// Simulate `devbox update` with no args: all packages are in the update list.
	devbox.packagesBeingUpdated = []*devpkg.Package{pkgA, pkgB}

	require.True(t, devbox.isBeingUpdated(pkgA))
	require.True(t, devbox.isBeingUpdated(pkgB))
}

func TestUpdateEmptyUpdateListSelectsNothing(t *testing.T) {
	devbox := devboxForTesting(t)

	helloPkg := devpkg.PackageFromStringWithDefaults("hello@1.2.3", nil)

	// packagesBeingUpdated is nil (default) — no package should be force-refreshed.
	require.False(t, devbox.isBeingUpdated(helloPkg))
}

func currentSystem(*testing.T) string {
	sys := nix.System() // NOTE: we could mock this too, if it helps.
	return sys
}

func TestFlakeUpdateRewritesLockEntry(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "github:numtide/flake-utils"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, devbox.lockfile)
	oldRev := "1111111111111111111111111111111111111111"
	newRev := "2222222222222222222222222222222222222222"
	existing := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + oldRev,
		LastModified: "2024-01-01T00:00:00Z",
	}
	resolved := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + newRev,
		LastModified: "2025-04-22T00:00:00Z",
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{raw: existing},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err)
	require.Equal(t, "github:numtide/flake-utils/"+newRev, lockfile.Packages[raw].Resolved)
	require.Equal(t, "2025-04-22T00:00:00Z", lockfile.Packages[raw].LastModified)
}

func TestFlakeUpdateStalenessGuardRejectsOlder(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "github:numtide/flake-utils"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, devbox.lockfile)
	newerRev := "2222222222222222222222222222222222222222"
	olderRev := "1111111111111111111111111111111111111111"
	existing := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + newerRev,
		LastModified: "2025-04-22T00:00:00Z",
	}
	resolved := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + olderRev,
		LastModified: "2024-01-01T00:00:00Z",
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{raw: existing},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err)
	// Entry must remain on the newer rev.
	require.Equal(t, "github:numtide/flake-utils/"+newerRev, lockfile.Packages[raw].Resolved)
}

// Regression: the staleness guard must not trigger when resolved.LastModified
// is empty (some nix error paths omit it). Missing == unknown, not older.
func TestFlakeUpdateAllowsMissingResolvedLastModified(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "github:numtide/flake-utils"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, devbox.lockfile)
	oldRev := "1111111111111111111111111111111111111111"
	newRev := "2222222222222222222222222222222222222222"
	existing := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + oldRev,
		LastModified: "2025-04-22T00:00:00Z",
	}
	resolved := &lock.Package{
		Resolved: "github:numtide/flake-utils/" + newRev,
		// LastModified deliberately empty.
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{raw: existing},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err)
	require.Equal(t, "github:numtide/flake-utils/"+newRev, lockfile.Packages[raw].Resolved)
}

func TestFlakeUpdateNoOpWhenResolvedUnchanged(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "github:numtide/flake-utils"
	devPkg := devpkg.PackageFromStringWithDefaults(raw, devbox.lockfile)
	rev := "1111111111111111111111111111111111111111"
	existing := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + rev,
		LastModified: "2024-01-01T00:00:00Z",
	}
	resolved := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + rev,
		LastModified: "2024-01-01T00:00:00Z",
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{raw: existing},
	}

	err := devbox.mergeResolvedPackageToLockfile(devPkg, resolved, lockfile)
	require.NoError(t, err)
	require.Same(t, existing, lockfile.Packages[raw], "entry should not be replaced on no-op update")
}

func TestShortRev(t *testing.T) {
	// GitHub refs only parse as "locked" (with a Rev) if the third path
	// component is a 40-char hex SHA. Anything shorter is treated as a ref
	// name, not a revision.
	longRev := "abc1234def56789012345678901234567890abcd"
	cases := []struct {
		in, want string
	}{
		{"github:numtide/flake-utils/" + longRev + "#pkg", "abc1234"},
		{"path:./local", ""},
		{"", ""},
		{"not a flake ref", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			require.Equal(t, tc.want, shortRev(tc.in))
		})
	}
}

func TestShortDate(t *testing.T) {
	require.Equal(t, "2025-04-22", shortDate("2025-04-22T14:30:00Z"))
	require.Equal(t, "", shortDate(""))
	require.Equal(t, "", shortDate("not a date"))
}

func TestDescribeFlakeUpdateFormats(t *testing.T) {
	oldRev := "abc1234def56789012345678901234567890abcd"
	newRev := "f4567890123456789abcdef012345678901234ab"
	oldPkg := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + oldRev + "#pkg",
		LastModified: "2024-11-01T00:00:00Z",
	}
	newPkg := &lock.Package{
		Resolved:     "github:numtide/flake-utils/" + newRev + "#pkg",
		LastModified: "2025-04-22T00:00:00Z",
	}
	require.Equal(t,
		"abc1234 -> f456789  (2024-11-01 → 2025-04-22)",
		describeFlakeUpdate(oldPkg, newPkg),
	)

	// Fallback to date-only when refs have no rev (e.g. path:).
	oldPath := &lock.Package{Resolved: "path:./x", LastModified: "2024-11-01T00:00:00Z"}
	newPath := &lock.Package{Resolved: "path:./x", LastModified: "2025-04-22T00:00:00Z"}
	require.Equal(t, "(2024-11-01 → 2025-04-22)", describeFlakeUpdate(oldPath, newPath))

	// Fallback to rev-only when dates missing.
	oldNoDate := &lock.Package{Resolved: "github:numtide/flake-utils/" + oldRev + "#pkg"}
	newNoDate := &lock.Package{Resolved: "github:numtide/flake-utils/" + newRev + "#pkg"}
	require.Equal(t, "abc1234 -> f456789", describeFlakeUpdate(oldNoDate, newNoDate))
}
