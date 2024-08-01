package plugins

import (
	"embed"
	"io/fs"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

//go:embed *.json */*
var builtIn embed.FS

func Builtins() ([]fs.DirEntry, error) {
	entries, err := builtIn.ReadDir(".")
	if err != nil {
		return nil, err
	}
	return lo.Filter(entries, func(e fs.DirEntry, _ int) bool {
		return !e.IsDir() && !strings.HasSuffix(e.Name(), ".go")
	}), nil
}

type BuiltIn struct{}

var builtInMap = map[*regexp.Regexp]string{
	regexp.MustCompile(`^(apache|apacheHttpd)$`):                       "apacheHttpd",
	regexp.MustCompile(`^(gradle|gradle_[0-9])$`):                      "gradle",
	regexp.MustCompile(`^(ghc|haskell\.compiler\.(.*))$`):              "haskell",
	regexp.MustCompile(`^mariadb(-embedded)?_?[0-9]*$`):                "mariadb",
	regexp.MustCompile(`^mysql?[0-9]*$`):                               "mysql",
	regexp.MustCompile(`^nodejs(-slim)?_?[0-9]*$`):                     "nodejs",
	regexp.MustCompile(`^php[0-9]*$`):                                  "php",
	regexp.MustCompile(`^python3[0-9]*Packages.pip$`):                  "pip",
	regexp.MustCompile(`^(\w*\.)?poetry$`):                             "poetry",
	regexp.MustCompile(`^postgresql(_[0-9]+)?$`):                       "postgresql",
	regexp.MustCompile(`^python[0-9]*(Full|Minimal|-full|-minimal)?$`): "python",
	regexp.MustCompile(`^redis$`):                                      "redis",
	regexp.MustCompile(`^j?ruby([0-9_]*[0-9]+)?$`):                     "ruby",
	regexp.MustCompile(`^valkey$`):										"valkey",
}

func BuiltInForPackage(pkgName string) ([]byte, error) {
	for re, name := range builtInMap {
		if re.MatchString(pkgName) {
			return builtIn.ReadFile(name + ".json")
		}
	}
	return builtIn.ReadFile(pkgName + ".json")
}

func (f *BuiltIn) FileContent(contentPath string) ([]byte, error) {
	return builtIn.ReadFile(contentPath)
}
