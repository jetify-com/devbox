// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"
	"text/tabwriter"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/spf13/cobra"

	"go.jetify.com/devbox/internal/boxcli/usererr"
	"go.jetify.com/devbox/internal/searcher"
	"go.jetify.com/devbox/internal/ux"
)

const trimmedVersionsLength = 10

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
			name, version, isVersioned := searcher.ParseVersionedPackage(query)
			if !isVersioned {
				client := searcher.Client()
				limit := trimmedVersionsLength + 1
				if flags.showAll {
					limit = 0
				}
				results, err := client.Search(cmd.Context(), query, limit)
				if err != nil {
					return err
				}
				return printSearchResults(
					cmd.OutOrStdout(), query, results, flags.showAll, client.Host(),
				)
			}
			packageVersion, err := searcher.Client().Resolve(name, version)
			if err != nil {
				// This is not ideal. Search service should return valid response we
				// can parse
				return usererr.WithUserMessage(err, "No results found for %q\n", query)
			}
			fmt.Fprintf(
				cmd.OutOrStdout(),
				"%s resolves to: %s@%s\n",
				query,
				packageVersion.Name,
				packageVersion.Version,
			)
			return nil
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
	results *searcher.SearchResults,
	showAll bool,
	host string,
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

	resultsAreTrimmed := false
	pkgs := results.Packages
	if !showAll && len(pkgs) > trimmedVersionsLength {
		resultsAreTrimmed = true
		pkgs = pkgs[:trimmedVersionsLength]
	}

	tableWriter := table.NewWriter()
	tableWriter.AppendHeader(table.Row{"Package", "Versions", "Platforms"})
	appendRow := func(name string, versions, systems []string) {
		versionStr := columnize(versions, 2)
		systemColumns := 1
		if strings.Count(versionStr, "\n")+1 < len(systems) {
			systemColumns = 2
		}
		tableWriter.AppendRow(table.Row{name, versionStr, columnize(systems, systemColumns)})
	}

	for _, pkg := range pkgs {
		var versions, systems []string
		shown := 0
		for _, pkgVersion := range pkg.Versions {
			if pkgVersion.Version == "" {
				continue
			}
			if !showAll && shown >= trimmedVersionsLength {
				resultsAreTrimmed = true
				break
			}
			shown++

			if len(versions) > 0 && !slices.Equal(systems, pkgVersion.Systems) {
				appendRow(pkg.Name, versions, systems)
				versions = nil
			}
			versions = append(versions, pkgVersion.Version)
			systems = pkgVersion.Systems
		}
		if len(versions) > 0 || shown == 0 {
			appendRow(pkg.Name, versions, systems)
		}
	}

	tableWriter.SetColumnConfigs([]table.ColumnConfig{
		{Number: 1, AutoMerge: true, VAlign: text.VAlignMiddle},
		{Number: 2, AlignHeader: text.AlignCenter},
		{Number: 3, AlignHeader: text.AlignCenter},
	})
	tableWriter.SetStyle(table.StyleLight)
	tableWriter.Style().Options.SeparateRows = true
	fmt.Fprintln(w, tableWriter.Render())

	if resultsAreTrimmed {
		fmt.Fprintln(w)
		ux.Fwarningf(
			w,
			"Showing top %d results and truncated versions. Use --show-all to "+
				"show all.\n\n",
			trimmedVersionsLength,
		)
	}
	fmt.Fprintf(w, "For more information go to: %s/\n", strings.TrimSuffix(host, "/"))

	return nil
}

func columnize(data []string, maxColumns int) string {
	columns := maxColumns
	if len(data) <= columns {
		columns = 1
	}

	buf := bytes.NewBufferString("")
	var versionsGroup []string
	writer := tabwriter.NewWriter(buf, 0, 0, 2, ' ', 0)
	for _, version := range data {
		if len(versionsGroup) == columns {
			_, _ = fmt.Fprintf(writer, "%s\n", strings.Join(versionsGroup, "\t"))
			versionsGroup = nil
		}
		versionsGroup = append(versionsGroup, version)
	}
	if len(versionsGroup) > 0 {
		_, _ = fmt.Fprintf(writer, "%s\n", strings.Join(versionsGroup, "\t"))
	}
	_ = writer.Flush()
	return strings.TrimSuffix(buf.String(), "\n")
}
