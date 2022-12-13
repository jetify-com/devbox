package mutagenbox

import (
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

const shimDirPath = ".config/devbox/ssh/shims"

func ShimDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.WithStack(err)
	}
	shimDir := filepath.Join(home, shimDirPath)
	return shimDir, nil
}
