package pkgsuggest

import (
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/dotnet"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/haskell"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/javascript"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/rust"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/zig"
)

var SUGGESTORS = []suggestors.Suggestor{
	&dotnet.Suggestor{},
	&haskell.Suggestor{},
	&javascript.Suggestor{},
	&rust.Suggestor{},
	&zig.Suggestor{},
}

func GetSuggestors(srcDir string) ([]string, error) {
	result := []string{}
	for _, sg := range SUGGESTORS {
		if sg.IsRelevant(srcDir) {
			result = append(result, sg.Packages()...)
		}
	}

	return result, nil
}
