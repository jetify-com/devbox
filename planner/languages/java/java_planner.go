// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package java

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/creekorful/mvnparser"
	"github.com/pkg/errors"
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
	return fileExists(pomXMLPath)
}

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	javaPkg := getJavaPackage(srcDir)
	return &plansdk.Plan{
		DevPackages: []string{
			javaPkg,
			defaultMaven,
		},
	}
}

func getJavaPackage(srcDir string) string {
	pomXMLPath := filepath.Join(srcDir, "pom.xml")
	javaVersion := parseJavaVersion(pomXMLPath)
	v, ok := jVersionMap[javaVersion]
	if ok {
		return v
	} else {
		return defaultJava
	}
}

func parseJavaVersion(pomXMLPath string) int {
	parsedVersion, err := parseXML(pomXMLPath)
	if err != nil {
		fmt.Printf("error parsing java version from pom file: %v", err)
		return 0
	}
	return parsedVersion
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func parseXML(pomXMLPath string) (int, error) {
	// read the XML file as a byte array.
	byteArray, err := os.ReadFile(pomXMLPath)
	if err != nil {
		return 0, nil
	}

	var project mvnparser.MavenProject
	// unmarshaling byteArray which contains our pom file content into 'project'
	xml.Unmarshal(byteArray, &project)
	compilerSourceVersion, ok := project.Properties["maven.compiler.source"]
	if ok {
		sourceVersion, err := strconv.Atoi(compilerSourceVersion)
		if err != nil {
			return 0, errors.WithStack(err)
		}
		return sourceVersion, nil
	}

	return 0, nil
}
