package initrec

import (
	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/initrec/recommenders/dotnet"
	"go.jetpack.io/devbox/internal/initrec/recommenders/golang"
	"go.jetpack.io/devbox/internal/initrec/recommenders/haskell"
	"go.jetpack.io/devbox/internal/initrec/recommenders/java"
	"go.jetpack.io/devbox/internal/initrec/recommenders/javascript"
	"go.jetpack.io/devbox/internal/initrec/recommenders/nginx"
	"go.jetpack.io/devbox/internal/initrec/recommenders/python"
	"go.jetpack.io/devbox/internal/initrec/recommenders/ruby"
	"go.jetpack.io/devbox/internal/initrec/recommenders/rust"
	"go.jetpack.io/devbox/internal/initrec/recommenders/zig"
)

func getRecommenders(srcDir string) []recommenders.Recommender {

	return []recommenders.Recommender{
		&dotnet.Recommender{SrcDir: srcDir},
		&golang.Recommender{SrcDir: srcDir},
		&haskell.Recommender{SrcDir: srcDir},
		&java.Recommender{SrcDir: srcDir},
		&javascript.Recommender{SrcDir: srcDir},
		&nginx.Recommender{SrcDir: srcDir},
		&python.RecommenderPip{SrcDir: srcDir},
		&python.RecommenderPoetry{SrcDir: srcDir},
		&ruby.Recommender{SrcDir: srcDir},
		&rust.Recommender{SrcDir: srcDir},
		&zig.Recommender{SrcDir: srcDir},
	}
}

func Get(srcDir string) ([]string, error) {

	result := []string{}
	for _, sg := range getRecommenders(srcDir) {
		if sg.IsRelevant() {
			result = append(result, sg.Packages()...)
		}
	}

	return result, nil
}
