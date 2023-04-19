// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package searcher

import (
	"fmt"
	"io"
	"strings"

	"github.com/samber/lo"
	"go.jetpack.io/devbox/internal/boxcli/usererr"
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

func GenLockedReferences(pkgs []string) ([]string, error) {
	c := NewClient()
	references := append([]string(nil), pkgs...) // copy
	for i, pkg := range pkgs {
		if name, version, found := strings.Cut(pkg, "@"); found {
			result, err := c.SearchVersion(name, version)
			if err != nil {
				return nil, err
			}
			if len(result.Results) == 0 {
				errorText := fmt.Sprintf("No results found for %q.", pkg)
				if len(result.Suggestions) > 0 && len(result.Suggestions[0].Packages) > 0 {
					versions := lo.Map(
						result.Suggestions[0].Packages,
						func(p *NixPackageInfo, _ int) string { return p.Version },
					)
					errorText += fmt.Sprintf(
						" Available versions %s",
						strings.Join(versions, ", "),
					)
				}
				return nil, usererr.New(errorText + "\n")
			}

			references[i] = fmt.Sprintf(
				"github:NixOS/nixpkgs/%s#%s",
				result.Results[0].Packages[0].NixpkgCommit,
				result.Results[0].Packages[0].AttributePath,
			)
		}
	}
	return references, nil
}
