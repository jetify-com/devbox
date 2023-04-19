// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"fmt"
	"io"
	"strings"

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
