// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/samber/lo"

	"go.jetpack.io/devbox/internal/redact"
	"go.jetpack.io/devbox/internal/searcher/model"
)

func SearchAndPrint(w io.Writer, query string) error {
	c := Client()
	result, err := c.Search(query)
	if err != nil {
		return redact.Errorf("failed to get search results: %v", redact.Safe(err))
	}
	if len(result.Packages) == 0 {
		fmt.Fprintf(w, "No results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(
		w,
		"Found %d+ results for %q:\n\n",
		result.NumResults,
		query,
	)

	for _, r := range result.Packages[:int(math.Min(10, float64(len(result.Packages))))] {
		versions := lo.Map(r.Versions, func(p model.PackageVersion, _ int) string {
			return p.Version
		})

		fmt.Fprintf(w, "* %s (%s)\n", r.Name, strings.Join(versions, ", "))
	}
	return nil
}

// ParseVersionedPackage checks if the given package is a versioned package
// (`python@3.10`) and returns its name and version
func ParseVersionedPackage(pkg string) (string, string, bool) {
	lastIndex := strings.LastIndex(pkg, "@")
	if lastIndex == -1 {
		return "", "", false
	}
	return pkg[:lastIndex], pkg[lastIndex+1:], true
}
