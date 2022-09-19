// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package java

import (
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

// jdk points to openJDK version 17. OpenJDK v18 is not yet available in nix packages
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
	plan := &plansdk.Plan{
		DevPackages: []string{
			defaultMaven,
		},
	}
	javaPkg, err := getJavaPackage(srcDir)
	if err != nil {
		return plan.WithError(err)
	}
	plan.DevPackages = append(plan.DevPackages, javaPkg)
	return plan
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
	var project mvnparser.MavenProject
	// parsing pom.xml and putting its content in 'project'
	err := cuecfg.ParseFile(pomXMLPath, &project)
	if err != nil {
		return 0, errors.WithMessage(err, "error parsing java version from pom file")
	}
	compilerSourceVersion, ok := project.Properties["maven.compiler.source"]
	if ok {
		sourceVersion, err := strconv.Atoi(compilerSourceVersion)
		if err != nil {
			return 0, errors.WithMessage(err, "error parsing java version from pom file")
		}
		return sourceVersion, nil
	}

	return 0, nil
}
