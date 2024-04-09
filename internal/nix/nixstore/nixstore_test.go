// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package nixstore

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"
)

// Some tests check their results against `nix path-info` output which is saved
// in the testdata directory. To update or regenerate the nix-path info output,
// run `go generate` and commit the results.

//go:generate sh -c "nix path-info --recursive --json /nix/store/mil5crms7gfpv03vjj094zz1igvapv6i-go-1.20.2 > testdata/mil5crms7gfpv03vjj094zz1igvapv6i-go-1.20.2.json"

func TestLocalStorePackage(t *testing.T) {
	if _, err := os.Stat("/nix/store/mil5crms7gfpv03vjj094zz1igvapv6i-go-1.20.2"); err != nil {
		t.Skip(`run "nix copy --from https://cache.nixos.org /nix/store/mil5crms7gfpv03vjj094zz1igvapv6i-go-1.20.2" to run this test`)
	}
	storePath := "/nix/store"
	local, err := Local(storePath)
	if err != nil {
		t.Fatalf("got error for local Nix store %s: %v", storePath, err)
	}
	pkg, err := local.Package("mil5crms7gfpv03vjj094zz1igvapv6i-go-1.20.2")
	if err != nil {
		t.Fatalf("got error querying package %s: %v", pkg, err)
	}
	checkDependencies(t, pkg, unmarshalNixPathInfoOutput(t, pkg))
}

func checkDependencies(t *testing.T, got *Package, nixPathInfos map[string][]string) {
	t.Helper()

	want, ok := nixPathInfos[got.StoreName]
	if !ok {
		t.Errorf("got unwanted package: %s", got)
	}
	if len(got.DirectDependencies) != len(want) {
		t.Fatalf("package %s has wrong number of dependencies:\ngot:  %v\nwant: %v",
			got, got.DirectDependencies, want)
	}
	sort.Slice(got.DirectDependencies, func(i, j int) bool {
		return got.DirectDependencies[i].StoreName < got.DirectDependencies[j].StoreName
	})
	for i, dep := range got.DirectDependencies {
		if dep.StoreName != want[i] {
			t.Fatalf("package %s has unwanted dependency %s:\ngot:  %v\nwant: %v",
				got, dep, got.DirectDependencies, want)
		}
		checkDependencies(t, dep, nixPathInfos)
	}
}

func unmarshalNixPathInfoOutput(t *testing.T, got *Package) map[string][]string {
	t.Helper()

	testdata := "testdata/" + got.StoreName + ".json"
	b, err := os.ReadFile(testdata)
	if err != nil {
		t.Fatalf("got error reading %s: %v", testdata, err)
	}

	var pathInfos []struct {
		Path       string
		References []string
	}
	if err := json.Unmarshal(b, &pathInfos); err != nil {
		t.Fatalf("got error unmarshalling %s: %v", testdata, err)
	}
	depsByPackage := make(map[string][]string, len(pathInfos))
	for _, pinfo := range pathInfos {
		refs := make([]string, 0, len(pinfo.References))
		for _, ref := range pinfo.References {
			if ref == pinfo.Path {
				continue
			}
			refs = append(refs, strings.TrimPrefix(ref, "/nix/store/"))
		}
		sort.Strings(refs)
		depsByPackage[strings.TrimPrefix(pinfo.Path, "/nix/store/")] = refs
	}
	return depsByPackage
}
