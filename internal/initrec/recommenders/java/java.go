// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package java

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"

	"github.com/creekorful/mvnparser"
	"github.com/pkg/errors"

	"go.jetpack.io/devbox/internal/cuecfg"
	"go.jetpack.io/devbox/internal/fileutil"
	"go.jetpack.io/devbox/internal/initrec/recommenders"
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

// misc. nix packages
const binUtils = "binutils"

// builder tool specific names
const (
	MavenType      = "maven"
	GradleType     = "gradle"
	mavenFileName  = "pom.xml"
	gradleFileName = "build.gradle"
)

// jdk nix packages
var jVersionMap = map[string]string{
	"8":  "jdk8",
	"11": "jdk11",
	"17": "jdk17_headless",
	"18": "jdk18_headless",
}

// default nix packages
const (
	defaultJava   = "jdk" // "jdk" points to openJDK version 17
	defaultMaven  = "maven"
	defaultGradle = "gradle"
)

type Recommender struct {
	SrcDir string
}

// implements interface recommenders.Recommender (compile-time check)
var _ recommenders.Recommender = (*Recommender)(nil)

func (r *Recommender) IsRelevant() bool {
	pomXMLPath := filepath.Join(r.SrcDir, mavenFileName)
	buildGradlePath := filepath.Join(r.SrcDir, gradleFileName)
	return fileutil.Exists(pomXMLPath) || fileutil.Exists(buildGradlePath)
}

func (r *Recommender) Packages() []string {
	builderTool, err := r.packageManager()
	if err != nil {
		return nil
	}
	devPackages, _ := r.devPackages(builderTool)
	// if err is not nil, devPackages will be nil
	return devPackages
}

func (r *Recommender) packageManager() (string, error) {
	pomXMLPath := filepath.Join(r.SrcDir, mavenFileName)
	buildGradlePath := filepath.Join(r.SrcDir, gradleFileName)
	if fileutil.Exists(pomXMLPath) {
		return MavenType, nil
	}
	if fileutil.Exists(buildGradlePath) {
		return GradleType, nil
	}
	return "", errors.New("could not locate a Maven or Gradle file")
}

func (r *Recommender) devPackages(builderTool string) ([]string, error) {
	javaPkg, err := getJavaPackage(r.SrcDir, builderTool)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	devPackagesMap := map[string][]string{
		MavenType: {
			defaultMaven,
			javaPkg,
			binUtils,
		},
		GradleType: {
			defaultGradle,
			javaPkg,
			binUtils,
		},
	}

	return devPackagesMap[builderTool], nil
}

func getJavaPackage(srcDir string, builderTool string) (string, error) {
	javaVersion, err := parseJavaVersion(srcDir, builderTool)
	if err != nil {
		return "", errors.WithStack(err)
	}
	v, ok := jVersionMap[javaVersion.Major()]
	if ok {
		return v, nil
	}
	return defaultJava, nil
}

func parseJavaVersion(srcDir string, builderTool string) (*plansdk.Version, error) {
	sourceVersion, _ := plansdk.NewVersion("0")

	if builderTool == MavenType {
		pomXMLPath := filepath.Join(srcDir, mavenFileName)
		var parsedPom mvnparser.MavenProject
		// parsing pom.xml and putting its content in 'project'
		err := cuecfg.ParseFile(pomXMLPath, &parsedPom)
		if err != nil {
			return nil, errors.WithMessage(err, "error parsing java version from pom file")
		}
		compilerSourceVersion, ok := parsedPom.Properties["maven.compiler.source"]
		if ok {
			sourceVersion, err = plansdk.NewVersion(compilerSourceVersion)
			if err != nil {
				return nil, errors.WithMessage(err, "error parsing java version from pom file")
			}
		}
	} else if builderTool == GradleType {
		buildGradlePath := filepath.Join(srcDir, gradleFileName)
		readFile, err := os.Open(buildGradlePath)
		if err != nil {
			return nil, errors.WithMessage(err, "error parsing java version from gradle file")
		}
		fileScanner := bufio.NewScanner(readFile)
		fileScanner.Split(bufio.ScanLines)
		// parsing gradle file line by line
		for fileScanner.Scan() {
			line := fileScanner.Text()
			if strings.Contains(line, "sourceCompatibility = ") {
				compilerSourceVersion := strings.TrimSpace(strings.Split(line, "=")[1])
				sourceVersion, err = plansdk.NewVersion(compilerSourceVersion)
				if err != nil {
					return nil, errors.WithMessage(err, "error parsing java version from gradle file")
				}
				break
			}
		}
		readFile.Close()
	}

	return sourceVersion, nil
}
