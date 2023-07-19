package plugins

import (
	"embed"
	"io/fs"
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

type BuiltIn struct {
}

func (f *BuiltIn) FileContent(contentPath string) ([]byte, error) {
	return builtIn.ReadFile(contentPath)
}
