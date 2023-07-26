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
