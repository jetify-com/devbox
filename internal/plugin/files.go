package plugin

import (
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"go.jetpack.io/devbox/internal/xdg"
	"go.jetpack.io/devbox/plugins"
)

func getConfigIfAny(pkg, projectDir string) (*config, error) {
	configFiles, err := plugins.BuiltIn.ReadDir(".")
	if err != nil {
		return nil, errors.WithStack(err)
	}

	xdgRuntimePath, err := setupXdgRuntimePath(projectDir)
	if err != nil {
		return nil, err
	}

	// Try to find perfect match first
	for _, file := range configFiles {
		if file.IsDir() || strings.HasSuffix(file.Name(), ".go") {
			continue
		}
		content, err := plugins.BuiltIn.ReadFile(file.Name())
		if err != nil {
			return nil, errors.WithStack(err)
		}

		cfg, err := buildConfig(pkg, projectDir, xdgRuntimePath, string(content))
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// if match regex is set we use it to check. Otherwise we assume it's a
		// perfect match
		if (cfg.Match != "" && !regexp.MustCompile(cfg.Match).MatchString(pkg)) ||
			(cfg.Match == "" && strings.Split(file.Name(), ".")[0] != pkg) {
			continue
		}
		return cfg, nil
	}
	return nil, nil
}

func getFileContent(contentPath string) ([]byte, error) {
	return plugins.BuiltIn.ReadFile(contentPath)
}

// Setup and return an XDG Runtime Path for the virtenv
//
// We use a symlink for this path, but continue to store the data in the
// <project-dir>/.devbox/virtenv directory for ease of inspection.
//
// The symlink enables the path to be short so that unix sockets can be placed
// in this directory. Unix sockets have legacy ~100 char limits for their paths.
//
// TODO savil. Calculate this once for each plugin, and reuse.
// It is currently hard to do since there is no Init() method for plugins
// in which we can do this.
func setupXdgRuntimePath(projectDir string) (string, error) {

	virtenvXdgRuntimePath, err := virtenvRuntimeLinkPath(projectDir)
	if err != nil {
		return "", err
	}
	virtenvPath, err := ensureVirtenvPath(projectDir)
	if err != nil {
		return "", errors.WithStack(err)
	}
	if err := os.Symlink(virtenvPath, virtenvXdgRuntimePath); err != nil && !errors.Is(err, os.ErrExist) {
		return "", errors.WithStack(err)
	}
	return virtenvXdgRuntimePath, nil
}

// virtenvPathHashLength is the length of the hash used in the virtenv-<hash> in virtualenvRuntimeLinkPath.
// The <hash> is derived from the projectDir
const virtenvPathHashLength = 10

// virtenvRuntimeLinkPath returns a path for the given project's virtenv's Runtime resources to live in.
// It strives to be XDG compliant, but will diverge if needed to be short enough to be used in unix sockets.
func virtenvRuntimeLinkPath(projectDir string) (string, error) {

	// deriveLinkName returns the name of the link to be used.
	// It is of the form virtenv-<hash> where <hash> is a short non-crypto hash of the projectDir.
	// This disambiguates devbox/virtenv directories for different projects.
	deriveLinkName := func(projectDir string) (string, error) {
		h := fnv.New32a()
		_, err := h.Write([]byte(projectDir))
		if err != nil {
			return "", errors.WithStack(err)
		}
		hashed := strconv.FormatUint(uint64(h.Sum32()), 10)
		if len(hashed) > virtenvPathHashLength { // the length is expected to be 10 but being defensive.
			hashed = hashed[:virtenvPathHashLength]
		}
		linkName := fmt.Sprintf("virtenv-%s", hashed)
		return linkName, nil
	}

	// deriveDirectory returns the directory to be used, subject to the maxDirLen.
	// The motivation for maxDirLen is that unix sockets have legacy ~100 char limits for their paths.
	// Some plugins use unix sockets, and we want to place them in the XDG Runtime Path.
	//
	// Implementation: use XDG_RUNTIME_DIR and fallback to /tmp/user-<uid> if maxDirLen is exceeded.
	deriveDirectory := func(maxDirLen int) (string, error) {

		xdgRuntimeDir, err := xdg.RuntimeSubpath("devbox")
		if err != nil {
			return "", errors.WithStack(err)
		}

		if len([]byte(xdgRuntimeDir)) < maxDirLen {
			if err := os.MkdirAll(xdgRuntimeDir, 0700); err != nil {
				return "", errors.WithStack(err)
			}
			return xdgRuntimeDir, nil
		}

		// TODO hook up to correct io.writer
		//ux.Fwarning("xdgRuntimeDir is too long for unix-socket paths, "+
		//	"falling back to /tmp/user-<uid>. xdgRuntime: %s\n", xdgRuntimeDir)

		// Fallback to /tmp/user-<uid>/devbox/ dir
		// We do not use os.TempDir() since it can be pretty long in MacOS.
		// /tmp is posix-compliant and expected to exist, and is short.
		dir := filepath.Join(fmt.Sprintf("/tmp/user-%d", os.Getuid()), "devbox")
		if err := os.MkdirAll(dir, 0700); err != nil {
			return "", errors.WithStack(err)
		}

		//if len([]byte(dir)) > maxDirLen {
		// TODO savil: hook up to correct io.writer
		// ux.Fwarning(writer,"XdgRuntimeDir fallback %s is too long for unix-socket paths, " dir)
		//}
		return dir, nil
	}

	linkName, err := deriveLinkName(projectDir)
	if err != nil {
		return "", err
	}

	// 104 is the max unix socket path length in macs. Linux is marginally more.
	// The full symlink path will be: {dir}/{linkName}/{pluginName}
	// So we calculate the max dir length as:
	maxDirLen := 104 /* max unix socket len */ - len(linkName) - 10 /* estimate for the pluginName */

	dir, err := deriveDirectory(maxDirLen)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, linkName), nil
}
