package nix

import (
	"path/filepath"
	"reflect"
	"testing"
)

type inputTestCase struct {
	pkg                string
	isFlake            bool
	name               string
	urlWithoutFragment string
	packages           []string
}

func TestInput(t *testing.T) {
	projectDir := "/tmp/my-project"
	cases := []inputTestCase{
		{
			pkg:                "path:path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-c7758d",
			urlWithoutFragment: "path://" + filepath.Join(projectDir, "path/to/my-flake"),
			packages:           []string{"my-package"},
		},
		{
			pkg:                "path:.#my-package",
			isFlake:            true,
			name:               "my-project-744eaa",
			urlWithoutFragment: "path://" + projectDir,
			packages:           []string{"my-package"},
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake#my-package",
			isFlake:            true,
			name:               "my-flake-773986",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			packages:           []string{"my-package"},
		},
		{
			pkg:                "path:/tmp/my-project/path/to/my-flake",
			isFlake:            true,
			name:               "my-flake-eaedce",
			urlWithoutFragment: "path:" + filepath.Join(projectDir, "path/to/my-flake"),
			packages:           []string{"default"},
		},
		{
			pkg:                "hello",
			isFlake:            false,
			name:               "hello-5d4140",
			urlWithoutFragment: "hello",
			packages:           []string{"hello"},
		},
	}

	for _, testCase := range cases {
		i := InputFromString(testCase.pkg, projectDir)
		if isFLake := i.IsFlake(); testCase.isFlake != isFLake {
			t.Errorf("IsFlake() = %v, want %v", isFLake, testCase.isFlake)
		}
		if name := i.Name(); testCase.name != name {
			t.Errorf("Name() = %v, want %v", name, testCase.name)
		}
		if urlWithoutFragment := i.URLWithoutFragment(); testCase.urlWithoutFragment != urlWithoutFragment {
			t.Errorf("URLWithoutFragment() = %v, want %v", urlWithoutFragment, testCase.urlWithoutFragment)
		}
		if packages := i.Packages(); !reflect.DeepEqual(testCase.packages, packages) {
			t.Errorf("Packages() = %v, want %v", packages, testCase.packages)
		}
	}
}
