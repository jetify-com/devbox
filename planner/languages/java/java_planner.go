// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package java

import (
	"fmt"
	"path/filepath"
	"strconv"

	"github.com/creekorful/mvnparser"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
)

type Planner struct{}

var jVersionMap = map[int]string{
	8:  "jdk8",
	11: "jdk11",
	17: "jdk17_headless",
}

// "jdk" points to openJDK version 17. OpenJDK v18 is not yet available in nix packages
const defaultJava = "jdk"
const defaultMaven = "maven"

// Implements interface Planner (compile-time check)
var _ plansdk.Planner = (*Planner)(nil)

func (p *Planner) Name() string {
	return "java.Planner"
}

func (p *Planner) IsRelevant(srcDir string) bool {
	// Checking for pom.xml (maven) only for now
	// TODO: add build.gradle file detection
	pomXMLPath := filepath.Join(srcDir, "pom.xml")
	return plansdk.FileExists(pomXMLPath)
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	// Creating an empty plan so that we can communicate an error to the user
	plan := &plansdk.Plan{
		DevPackages: []string{
			defaultMaven,
		},
	}
	javaPkg, err := getJavaPackage(srcDir)
	if err != nil {
		return plan.WithError(err)
	}
	startCommand, err := p.startCommand(srcDir)
	if err != nil {
		return plan.WithError(err)
	}
	installStage := p.installCommand(srcDir)
	return &plansdk.Plan{
		DevPackages: []string{
			defaultMaven,
			javaPkg,
		},
		RuntimePackages: []string{
			defaultMaven,
			javaPkg,
		},
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    installStage,
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    startCommand,
		},
	}
}

// This method is added because we plan to differentiate Gradle and Maven.
// Otherwise, we could just assign the value without calling this.
func (p *Planner) installCommand(srcDir string) string {
	// TODO: Add support for Gradle install command
	return "mvn clean install"
}

func (p *Planner) startCommand(srcDir string) (string, error) {
	pomXMLPath := fmt.Sprintf("%s/pom.xml", srcDir)
	var parsedPom mvnparser.MavenProject
	err := cuecfg.ParseFile(pomXMLPath, &parsedPom)
	if err != nil {
		return "", errors.WithMessage(err, "error parsing the pom file")
	}
	return fmt.Sprintf("java -jar target/%s-%s.jar", parsedPom.ArtifactId, parsedPom.Version), nil
}

func getJavaPackage(srcDir string) (string, error) {
	pomXMLPath := filepath.Join(srcDir, "pom.xml")
	javaVersion, err := parseJavaVersion(pomXMLPath)
	if err != nil {
		return "", errors.WithStack(err)
	}
	v, ok := jVersionMap[javaVersion]
	if ok {
		return v, nil
	} else {
		return defaultJava, nil
	}
}

func parseJavaVersion(pomXMLPath string) (int, error) {
	var parsedPom mvnparser.MavenProject
	// parsing pom.xml and putting its content in 'project'
	err := cuecfg.ParseFile(pomXMLPath, &parsedPom)
	if err != nil {
		return 0, errors.WithMessage(err, "error parsing java version from pom file")
	}
	compilerSourceVersion, ok := parsedPom.Properties["maven.compiler.source"]
	if ok {
		sourceVersion, err := strconv.Atoi(compilerSourceVersion)
		if err != nil {
			return 0, errors.WithMessage(err, "error parsing java version from pom file")
		}
		return sourceVersion, nil
	}

	return 0, nil
}
