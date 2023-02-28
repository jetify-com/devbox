package nix

import (
	"io"
	"strings"
)

var packageInstallIgnore = []string{
	`replacing old 'devbox-development'`,
	`installing 'devbox-development'`,
}

type PackageInstallWriter struct {
	io.Writer
}

func (fw *PackageInstallWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" && !fw.ignore(line) {
			_, err = io.WriteString(fw.Writer, "\t"+line+"\n")
			if err != nil {
				return
			}
		}
	}
	return len(p), nil
}

func (*PackageInstallWriter) ignore(line string) bool {
	for _, filter := range packageInstallIgnore {
		if strings.Contains(line, filter) {
			return true
		}
	}
	return false
}
