// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package java

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/creekorful/mvnparser"
	"github.com/pkg/errors"
	"go.jetpack.io/devbox/cuecfg"
	"go.jetpack.io/devbox/planner/plansdk"
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

func (p *Planner) GetPlan(srcDir string) *plansdk.Plan {
	// Creating an empty plan so that we can communicate an error to the user
	plan := &plansdk.Plan{
		DevPackages: []string{},
	}

	builderTool, err := p.packageManager(srcDir)
	if err != nil {
		return plan.WithError(err)
	}

	devPackages, err := p.devPackages(srcDir, builderTool)
	if err != nil {
		return plan.WithError(err)
	}

	startCommand, err := p.startCommand(srcDir, builderTool)
	if err != nil {
		return plan.WithError(err)
	}

	runtimePackages := p.runtimePackages(builderTool)
	installCommand := p.installCommand(builderTool)
	buildCommand := p.buildCommand()

	return &plansdk.Plan{
		DevPackages:     devPackages,
		RuntimePackages: runtimePackages,
		InstallStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    installCommand,
		},
		BuildStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    buildCommand,
		},
		StartStage: &plansdk.Stage{
			InputFiles: []string{"."},
			Command:    startCommand,
		},
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

func (p *Planner) runtimePackages(builderTool string) []string {
	return []string{
		binUtils,
	}
}

// This method is added because we plan to differentiate Gradle and Maven.
// Otherwise, we could just assign the value without calling this.
func (p *Planner) installCommand(builderTool string) string {
	installCommandMap := map[string]string{
		MavenType:  "mvn clean install",
		GradleType: "./gradlew build",
	}
	return installCommandMap[builderTool]
}

func (p *Planner) buildCommand() string {
	return "jlink --verbose" +
		" --add-modules ALL-MODULE-PATH" +
		" --strip-debug" +
		" --no-man-pages" +
		" --no-header-files" +
		" --compress=2" +
		" --output ./customjre"
}

func (p *Planner) startCommand(srcDir string, builderTool string) (string, error) {
	if builderTool == MavenType {
		pomXMLPath := fmt.Sprintf("%s/%s", srcDir, mavenFileName)
		var parsedPom mvnparser.MavenProject
		err := cuecfg.ParseFile(pomXMLPath, &parsedPom)
		if err != nil {
			return "", errors.WithMessage(err, "error parsing the pom file")
		}
		return fmt.Sprintf("./customjre/bin/java -jar target/%s-%s.jar", parsedPom.ArtifactId, parsedPom.Version), nil
	} else if builderTool == GradleType {
		return "export JAVA_HOME=./customjre && ./gradlew run", nil
	}
	return "", nil
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
