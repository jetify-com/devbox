// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package initrec

import (
	"github.com/samber/lo"
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
	// Using a map of string-bool instead of array of strings to prevent duplication
	result := map[string]bool{}
	for _, sg := range getRecommenders(srcDir) {
		if sg.IsRelevant() {
			for _, pkg := range sg.Packages() {
				result[pkg] = true
			}
		}
	}
	// TODO: check for already installed packages
	return lo.Keys(result), nil
}
