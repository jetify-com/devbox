package impl

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
)

func TestUpdateNewPackageIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	devPkg := &devpkg.Package{
		Raw: raw,
	}
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{}, // empty
	}

	err := devbox.mergeResolvedPackageToLockfile(context.Background(), devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
}

func TestUpdateNewCurrentSysInfoIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys := "system1"
	devPkg := &devpkg.Package{
		Raw: raw,
	}
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys: {
				StorePath:   "store_path1",
				CAStorePath: "ca_path1",
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Version:  "1",
				// No system infos.
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(context.Background(), devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys)
	require.Equal(t, "store_path1", lockfile.Packages[raw].Systems[sys].StorePath)
}

func TestUpdateNewSysInfoIsAdded(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys1 := "system1" // current system
	sys2 := "system2"
	devPkg := &devpkg.Package{
		Raw: raw,
	}
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys1: {
				StorePath:   "store_path1",
				CAStorePath: "ca_path1",
			},
			sys2: {
				StorePath: "store_path2",
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Version:  "1",
				Systems: map[string]*lock.SystemInfo{
					sys1: {
						StorePath:   "store_path1",
						CAStorePath: "ca_path1",
					},
					// Missing sys2
				},
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(context.Background(), devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys1)
	require.Contains(t, lockfile.Packages[raw].Systems, sys2)
	require.Equal(t, "store_path2", lockfile.Packages[raw].Systems[sys2].StorePath)
}

func TestUpdateOtherSysInfoIsReplaced(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys1 := "system1" // current system
	sys2 := "system2"
	devPkg := &devpkg.Package{
		Raw: raw,
	}
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys1: {
				StorePath:   "store_path1",
				CAStorePath: "ca_path1",
			},
			sys2: {
				StorePath: "store_path2",
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Version:  "1",
				Systems: map[string]*lock.SystemInfo{
					sys1: {
						StorePath:   "store_path1",
						CAStorePath: "ca_path1",
					},
					sys2: {
						StorePath:   "mismatching_store_path",
						CAStorePath: "ca_path2",
					},
				},
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(context.Background(), devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys1)
	require.Contains(t, lockfile.Packages[raw].Systems, sys2)
	require.Equal(t, "store_path1", lockfile.Packages[raw].Systems[sys1].StorePath)
	require.Equal(t, "store_path2", lockfile.Packages[raw].Systems[sys2].StorePath)
	require.Empty(t, lockfile.Packages[raw].Systems[sys2].CAStorePath)
}

func TestUpdateCAPathIsNotReplaced(t *testing.T) {
	devbox := devboxForTesting(t)

	raw := "hello@1.2.3"
	sys1 := "system1" // current system
	sys2 := "system2"
	devPkg := &devpkg.Package{
		Raw: raw,
	}
	resolved := &lock.Package{
		Resolved: "resolved-flake-reference",
		Systems: map[string]*lock.SystemInfo{
			sys1: {
				StorePath:   "store_path1",
				CAStorePath: "ca_path1",
			},
			sys2: {
				StorePath: "store_path2",
				// No CAPath here because this is not the current system.
			},
		},
	}
	lockfile := &lock.File{
		Packages: map[string]*lock.Package{
			raw: {
				Resolved: "resolved-flake-reference",
				Version:  "1",
				Systems: map[string]*lock.SystemInfo{
					sys1: {
						StorePath:   "store_path1",
						CAStorePath: "ca_path1",
					},
					sys2: {
						StorePath:   "store_path2",
						CAStorePath: "ca_path2", // we already have CAPath for this system; it should not be replaced
					},
				},
			},
		},
	}

	err := devbox.mergeResolvedPackageToLockfile(context.Background(), devPkg, resolved, lockfile)
	require.NoError(t, err, "update failed")

	require.Contains(t, lockfile.Packages, raw)
	require.Contains(t, lockfile.Packages[raw].Systems, sys1)
	require.Contains(t, lockfile.Packages[raw].Systems, sys2)
	require.Equal(t, "store_path2", lockfile.Packages[raw].Systems[sys2].StorePath)
	require.Equal(t, "ca_path2", lockfile.Packages[raw].Systems[sys2].CAStorePath)
}
