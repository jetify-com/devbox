package nixstore

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"io"
	"path"
)

//go:embed packages.json.gz
var packagesJSON []byte

var pkgByAttrPath map[string]indexedPackage

type indexedPackage struct {
	Out   string
	Paths []string
}

func buildSearchIndex() error {
	index := struct {
		Packages map[string]indexedPackage
	}{}
	r, err := gzip.NewReader(bytes.NewReader(packagesJSON))
	if err != nil {
		return err
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}

	pkgByAttrPath = make(map[string]indexedPackage, len(index.Packages))
	for _, info := range index.Packages {
		for _, attrPath := range info.Paths {
			pkgByAttrPath[attrPath] = info
		}
	}
	return nil
}

func SearchExact(attrPath string) (string, error) {
	if pkgByAttrPath == nil {
		buildSearchIndex()
	}
	return path.Base(pkgByAttrPath[attrPath].Out), nil
}
