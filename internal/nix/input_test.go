package nix

import (
	"fmt"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/samber/lo"
)

type inputTestCase struct {
	pkg                string
	isFlake            bool
	name               string
	urlWithoutFragment string
	packageName        string
}

func TestInput(t *testing.T) {
	projectDir := "/tmp/my-project"
	cases := []inputTestCase{
		{
			pkg:                "path:path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-c7758d",
			urlWithoutFragment: "path://" + filepath.Join(projectDir, "path/to/my-flake"),
			packageName:        "packages.x86_64-darwin.my-package",
		},
		{
			pkg:                "path:.#my-package",
			isFlake:            true,
			name:               "my-project-744eaa",
			urlWithoutFragment: "path://" + projectDir,
			packageName:        "packages.x86_64-darwin.my-package",
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-773986",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			packageName:        "packages.x86_64-darwin.my-package",
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake",
			isFlake:            true,
			name:               "my-flake-eaedce",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			packageName:        "packages.x86_64-darwin.default",
		},
		{
			pkg:                "hello",
			isFlake:            false,
			name:               "hello-5d4140",
			urlWithoutFragment: "hello",
			packageName:        "hello",
		},
		{
			pkg:                "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
			isFlake:            true,
			name:               "gh-nixos-nixpkgs-5233fd2ba76a3accb5aaa999c00509a11fd0793c",
			urlWithoutFragment: "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c",
			packageName:        "packages.x86_64-darwin.hello",
		},
		{
			pkg:                "github:F1bonacc1/process-compose",
			isFlake:            true,
			name:               "gh-F1bonacc1-process-compose",
			urlWithoutFragment: "github:F1bonacc1/process-compose",
			packageName:        "packages.x86_64-darwin.default",
		},
	}

	for _, testCase := range cases {
		i := testInputFromString(testCase.pkg, projectDir)
		if isFLake := i.IsFlake(); testCase.isFlake != isFLake {
			t.Errorf("IsFlake() = %v, want %v", isFLake, testCase.isFlake)
		}
		if name := i.Name(); testCase.name != name {
			t.Errorf("Name() = %v, want %v", name, testCase.name)
		}
		if urlWithoutFragment := i.urlWithoutFragment(); testCase.urlWithoutFragment != urlWithoutFragment {
			t.Errorf("URLWithoutFragment() = %v, want %v", urlWithoutFragment, testCase.urlWithoutFragment)
		}
		if packages := i.Package(); !reflect.DeepEqual(testCase.packageName, packages) {
			t.Errorf("Package() = %v, want %v", packages, testCase.packageName)
		}
	}
}

type testInput struct {
	Input
}

type lockfile struct {
	projectDir string
}

func (lockfile) ConfigHash() (string, error) {
	return "", nil
}

func (l *lockfile) ProjectDir() string {
	return l.projectDir
}

func (lockfile) IsVersionedPackage(pkg string) bool {
	return false
}

func (lockfile) Resolve(pkg string) (string, error) {
	return "", nil
}

func testInputFromString(s, projectDir string) *testInput {
	return lo.ToPtr(testInput{Input: *InputFromString(s, &lockfile{projectDir})})
}

func (i *testInput) Package() string {
	if i.IsFlake() {
		return fmt.Sprintf(
			"packages.x86_64-darwin.%s",
			lo.Ternary(i.Fragment != "", i.Fragment, "default"),
		)
	}
	return i.String()
}
