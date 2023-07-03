// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"fmt"
	"io"
	"math"
	"strings"

	"github.com/spf13/cobra"

	"go.jetpack.io/devbox/internal/searcher"
	"go.jetpack.io/devbox/internal/searcher/model"
)

type searchCmdFlags struct {
	showAll bool
}

func searchCmd() *cobra.Command {
	flags := &searchCmdFlags{}
	command := &cobra.Command{
		Use:   "search <pkg>",
		Short: "Search for nix packages",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := args[0]
			results, err := searcher.Client().Search(query)
			if err != nil {
				return err
			}
			return printSearchResults(cmd.OutOrStdout(), query, results, flags.showAll)
		},
	}

	command.Flags().BoolVar(
		&flags.showAll, "show-all", false,
		"show all available templates",
	)

	return command
}

func printSearchResults(
	w io.Writer,
	query string,
	results *model.SearchResults,
	showAll bool,
) error {
	if len(results.Packages) == 0 {
		fmt.Fprintf(w, "No results found for %q\n", query)
		return nil
	}
	fmt.Fprintf(
		w,
		"Found %d+ results for %q:\n\n",
		results.NumResults,
		query,
	)

	pkgs := results.Packages
	if !showAll && len(pkgs) > 10 {
		fmt.Fprint(
			w,
			"Showing top 10 results. Use --show-all to show all.\n\n",
		)
		pkgs = results.Packages[:int(math.Min(10, float64(len(results.Packages))))]
	}

	for _, r := range pkgs {
		nonEmptyVersions := []string{}
		for _, v := range r.Versions {
			if v.Version != "" {
				nonEmptyVersions = append(nonEmptyVersions, v.Version)
			}
		}

		versionString := ""
		if len(nonEmptyVersions) > 0 {
			versionString = fmt.Sprintf(" (%s)", strings.Join(nonEmptyVersions, ", "))
		}
		fmt.Fprintf(w, "* %s %s\n", r.Name, versionString)
	}
	return nil
}
