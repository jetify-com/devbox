// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
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
	"go.jetpack.io/devbox/internal/planner/plansdk"
)

type Planner struct{}

// jdk nix packages
var jVersionMap = map[string]string{
	"8":  "jdk8",
	"11": "jdk11",
	"17": "jdk17_headless",
}

// default nix packages
const (
	defaultJava   = "jdk" // "jdk" points to openJDK version 17. OpenJDK v18 is not yet available in nix packages
	defaultMaven  = "maven"
	defaultGradle = "gradle"
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

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "java.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	pomXMLPath := filepath.Join(srcDir, mavenFileName)
	buildGradlePath := filepath.Join(srcDir, gradleFileName)
	return plansdk.FileExists(pomXMLPath) || plansdk.FileExists(buildGradlePath)
}

func (p *Planner) GetShellPlan(srcDir string) *plansdk.ShellPlan {
	builderTool, err := p.packageManager(srcDir)
	if err != nil {
		return &plansdk.ShellPlan{}
	}
	devPackages, err := p.devPackages(srcDir, builderTool)
	if err != nil {
		return &plansdk.ShellPlan{}
	}

	return &plansdk.ShellPlan{
		DevPackages: devPackages,
	}
}

func (p *Planner) packageManager(srcDir string) (string, error) {
	pomXMLPath := filepath.Join(srcDir, mavenFileName)
	buildGradlePath := filepath.Join(srcDir, gradleFileName)
	if plansdk.FileExists(pomXMLPath) {
		return MavenType, nil
	} else if plansdk.FileExists(buildGradlePath) {
		return GradleType, nil
	} else {
		return "", errors.New("Could not locate a Maven or Gradle file.")
	}
}

func (p *Planner) devPackages(srcDir string, builderTool string) ([]string, error) {
	javaPkg, err := getJavaPackage(srcDir, builderTool)
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
	} else {
		return defaultJava, nil
	}
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
