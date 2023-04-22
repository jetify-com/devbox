package plugin

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/ux"
	"go.jetpack.io/devbox/internal/xdg"
)

// Create and return a path of a symlink to the virtenv directory.
//
// The symlink is stored in an XDG_STATE_HOME location, and links to
// the <project-dir>/.devbox/virtenv directory. We store data in the virtenv directory
// for ease of inspection.
//
// The symlink enables the path to be short so that unix sockets can be placed
// in this directory. Unix sockets have legacy ~100 char limits for their paths.
func createVirtenvSymlink(w io.Writer, projectDir string) (string, error) {

	symlinkPath, err := virtenvSymlinkPath(projectDir)
	if err != nil {
		return "", err
	}

	// maxSymlinkPathLen is the expected max length of the symlink path that could support
	// unix sockets being created within by some plugins (e.g. postgres).
	//
	// 104 is the max unix socket path length in macs. Linux is marginally more.
	// The full symlink path will be of the form: {symlinkPath}/{pluginName}
	// So we calculate the max dir length as:
	maxSymlinkPathLen := 104 /* max unix socket len */ - 10 /* estimate for the pluginName */
	if len(symlinkPath) > maxSymlinkPathLen {
		ux.Fwarning(w, "Virtenv's symlink (%s) in XDG_STATE_HOME (%s) "+
			"is longer than 104 characters. If a plugin you are using uses a unix-socket, then "+
			"it may not work. Consider changing XDG_STATE_HOME to a shorter path for devbox.",
			symlinkPath, xdg.StateSubpath("devbox"))
	}

	// Ensure the symlink path's directory exists
	if err := os.MkdirAll(filepath.Dir(symlinkPath), 0700); err != nil {
		return "", errors.WithStack(err)
	}

	// Create the symlink
	virtenvPath := filepath.Join(projectDir, VirtenvPath)
	if err := os.Symlink(virtenvPath, symlinkPath); err != nil && !errors.Is(err, fs.ErrExist) {
		return "", errors.WithStack(err)
	}
	return symlinkPath, nil
}

// virtenvPathHashLength is the length of the hash used in the v-<hash> in virtenvSymlinkPath.
const virtenvPathHashLength = 5

// virtenvSymlinkPath returns a path for a project's virtenv resources to live in.
func virtenvSymlinkPath(projectDir string) (string, error) {

	// hashProjectDir returns a hash of the projectDir, of length maxLen.
	hashProjectDir := func(projectDir string, maxLen int) (string, error) {

		h := md5.New()
		_, err := h.Write([]byte(projectDir))
		if err != nil {
			return "", errors.WithStack(err)
		}
		hashed := hex.EncodeToString(h.Sum(nil)[:])
		if len(hashed) > maxLen {
			hashed = hashed[:maxLen]
		}
		return hashed, nil
	}

	hashed, err := hashProjectDir(projectDir, virtenvPathHashLength)
	if err != nil {
		return "", err
	}
	// linkName is of the form v-<hash of project dir>.
	// This disambiguates devbox/virtenv directories for different projects.
	linkName := fmt.Sprintf("v-%s", hashed)

	return filepath.Join(xdg.StateSubpath("devbox"), linkName), nil
}
