package pkgsuggest

import (
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors"
	"go.jetpack.io/devbox/internal/pkgsuggest/suggestors/javascript"
)

var SUGGESTORS = []suggestors.Suggestor{
	&javascript.Suggestor{},
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
