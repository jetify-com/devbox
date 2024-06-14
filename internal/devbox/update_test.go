package devbox

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.jetpack.io/devbox/internal/devpkg"
	"go.jetpack.io/devbox/internal/lock"
	"go.jetpack.io/devbox/internal/nix"
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

func currentSystem(*testing.T) string {
	sys := nix.System() // NOTE: we could mock this too, if it helps.
	return sys
}
