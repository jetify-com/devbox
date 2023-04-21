// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/pkg/errors"
	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/redact"
)

func SearchAndPrint(w io.Writer, query string) error {
	c := NewClient()
	result, err := c.Search(query)
	if err != nil {
		return redact.Errorf("error getting search results: %v", redact.Safe(err))
	}
	if len(result.Results) == 0 {
		fmt.Fprintf(w, "No results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(
		w,
		"Found %d+ results for %q:\n\n",
		result.Metadata.TotalResults,
		query,
	)

	for _, r := range result.Results {
		versions := lo.Map(r.Packages, func(p *NixPackageInfo, _ int) string {
			return p.Version
		})

		fmt.Fprintf(w, "* %s (%s)\n", r.Name, strings.Join(versions, ", "))
	}
	return nil
}

func Exists(name, version string) (bool, error) {
	c := NewClient()
	result, err := c.SearchVersion(name, version)
	if err != nil {
		return false, err
	}
	return len(result.Results) > 0, nil
}

func FlakeURL(name, version string) string {
	return fmt.Sprintf(
		"https://search.devbox.sh/%s/%s.tar.gz",
		url.PathEscape(name),
		url.PathEscape(version),
	)
}

func URLIsDevboxPackage(url string) bool {
	return strings.HasPrefix(url, "https://search.devbox.sh/")
}

func GetNameAndVersionFromPath(path string) (string, string, error) {
	path = strings.TrimSuffix(path, ".tar.gz")
	parts := lo.Filter(
		strings.Split(strings.TrimSpace(path), "/"),
		func(s string, _ int) bool {
			return strings.TrimSpace(s) != ""
		},
	)
	if len(parts) != 2 {
		return "", "", errors.New("invalid path")
	}

	return parts[0], parts[1], nil
}
