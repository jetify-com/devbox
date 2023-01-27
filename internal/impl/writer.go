package impl

import (
	"io"
	"strings"
)

var nixPackageInstallIgnore = []string{
	`replacing old 'devbox-development'`,
	`installing 'devbox-development'`,
}

type nixPackageInstallWriter struct {
	w      io.Writer
	indent string
}

func (fw *nixPackageInstallWriter) Write(p []byte) (n int, err error) {
	lines := strings.Split(string(p), "\n")
	for _, line := range lines {
		if line != "" && !fw.ignore(line) {
			_, err = io.WriteString(fw.w, fw.indent+line+"\n")
			if err != nil {
				return
			}
		}
	}
	return len(p), nil
}

func (*nixPackageInstallWriter) ignore(line string) bool {
	for _, filter := range nixPackageInstallIgnore {
		if strings.Contains(line, filter) {
			return true
		}
	}
	return false
}
