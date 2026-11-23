// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package boxcli

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"go.jetify.com/devbox/internal/searcher"
)

const testHost = "https://www.nixsearch.com"

func searchVersion(version string, systems ...string) searcher.SearchRelease {
	return searcher.SearchRelease{Version: version, Systems: systems}
}

func TestPrintSearchResultsGroupsVersionsBySystems(t *testing.T) {
	results := &searcher.SearchResults{
		NumResults: 1,
		Packages: []searcher.Package{{
			Name: "ripgrep",
			Versions: []searcher.SearchRelease{
				searchVersion("15.0.0", "aarch64-darwin", "x86_64-linux"),
				searchVersion("14.1.1", "aarch64-darwin", "x86_64-linux"),
				searchVersion("12.1.1", "x86_64-linux"),
			},
		}},
	}

	var buf bytes.Buffer
	if err := printSearchResults(&buf, "ripgrep", results, false, testHost); err != nil {
		t.Fatal(err)
	}
	want := `Found 1+ results for "ripgrep":

┌─────────┬──────────┬────────────────┐
│ PACKAGE │ VERSIONS │    PLATFORMS   │
├─────────┼──────────┼────────────────┤
│ ripgrep │ 15.0.0   │ aarch64-darwin │
│         │ 14.1.1   │ x86_64-linux   │
│         ├──────────┼────────────────┤
│         │ 12.1.1   │ x86_64-linux   │
└─────────┴──────────┴────────────────┘
For more information go to: https://www.nixsearch.com/
`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("printSearchResults() mismatch (-want +got):\n%s", diff)
	}
}

func TestPrintSearchResultsTrimsVersions(t *testing.T) {
	pkg := searcher.Package{Name: "go"}
	for i := range trimmedVersionsLength + 1 {
		pkg.Versions = append(pkg.Versions, searchVersion(fmt.Sprintf("1.%d.0", i), "x86_64-linux"))
	}
	results := &searcher.SearchResults{NumResults: 1, Packages: []searcher.Package{pkg}}

	var buf bytes.Buffer
	if err := printSearchResults(&buf, "go", results, false, testHost); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); strings.Contains(out, "1.10.0") || !strings.Contains(out, "Use --show-all") {
		t.Errorf("versions are not trimmed:\n%s", out)
	}

	buf.Reset()
	if err := printSearchResults(&buf, "go", results, true, testHost); err != nil {
		t.Fatal(err)
	}
	if out := buf.String(); !strings.Contains(out, "1.10.0") || strings.Contains(out, "Use --show-all") {
		t.Errorf("--show-all trimmed versions:\n%s", out)
	}
}

func TestPrintSearchResultsKeepsPackagesWithoutVersions(t *testing.T) {
	results := &searcher.SearchResults{
		NumResults: 2,
		Packages: []searcher.Package{
			{Name: "1.0", Versions: []searcher.SearchRelease{searchVersion("1.0", "x86_64-linux")}},
			{Name: "empty"},
		},
	}

	var buf bytes.Buffer
	if err := printSearchResults(&buf, "q", results, false, "http://localhost:8080/"); err != nil {
		t.Fatal(err)
	}
	want := `Found 2+ results for "q":

┌─────────┬──────────┬──────────────┐
│ PACKAGE │ VERSIONS │   PLATFORMS  │
├─────────┼──────────┼──────────────┤
│ 1.0     │ 1.0      │ x86_64-linux │
├─────────┼──────────┼──────────────┤
│ empty   │          │              │
└─────────┴──────────┴──────────────┘
For more information go to: http://localhost:8080/
`
	if diff := cmp.Diff(want, buf.String()); diff != "" {
		t.Errorf("printSearchResults() mismatch (-want +got):\n%s", diff)
	}
}

func TestColumnizeAlignsColumns(t *testing.T) {
	got := columnize([]string{"1.10.0", "1.9.0", "1.8.10", "1.7.0", "1.6.0"}, 2)
	want := "1.10.0  1.9.0\n1.8.10  1.7.0\n1.6.0"
	if got != want {
		t.Errorf("columnize() = %q, want %q", got, want)
	}
}
