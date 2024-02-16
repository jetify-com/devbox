//nolint:varnamelen
package devconfig

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/assert"
	"github.com/tailscale/hujson"
	"golang.org/x/tools/txtar"
)

/*
The tests in this file use txtar to define test input and expected output.
This makes the JSON a lot easier to read vs. defining it in variables or structs
with weird indentation.

Tests begin by defining their JSON with:

  in, want := parseConfigTxtarTest(t, `an optional comment that will be logged with t.Log
  -- in --
  { }
  -- want --
  { "packages": { "go": "latest" } }`)
*/

func parseConfigTxtarTest(t *testing.T, test string) (in Config, want []byte) {
	t.Helper()

	ar := txtar.Parse([]byte(test))
	if comment := strings.TrimSpace(string(ar.Comment)); comment != "" {
		t.Log(comment)
	}
	for _, f := range ar.Files {
		switch f.Name {
		case "in":
			var err error
			in, err = loadBytes(f.Data)
			if err != nil {
				t.Fatalf("input devbox.json is invalid: %v\n%s", err, f.Data)
			}

		case "want":
			want = f.Data
		}
	}
	return in, want
}

func optBytesToStrings() cmp.Option {
	return cmp.Transformer("bytesToStrings", func(b []byte) string {
		return string(b)
	})
}

func optParseHujson() cmp.Option {
	f := func(b []byte) map[string]any {
		gotMin, err := hujson.Minimize(b)
		if err != nil {
			return nil
		}
		var m map[string]any
		if err := json.Unmarshal(gotMin, &m); err != nil {
			return nil
		}
		return m
	}
	return cmp.Transformer("parseHujson", f)
}

func TestNoChanges(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `a config that's loaded and saved without any changes should have unchanged json
-- in --
{ "packages": { "go": "latest" } }
-- want --
{ "packages": { "go": "latest" } }`)

	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageEmptyConfig(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{}
-- want --
{
  "packages": {
    "go": "latest"
  }
}`)

	in.FilePackages().Add("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageEmptyConfigWhitespace(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{

}
-- want --
{
  "packages": {
    "go": "latest"
  }
}`)

	in.FilePackages().Add("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageEmptyConfigComment(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
// Comment
{}
-- want --
// Comment
{
  "packages": {
    "go": "latest",
  },
}`)

	in.FilePackages().Add("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageNull(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{ "packages": null }
-- want --
{
  "packages": {
    "go": "latest"
  }
}`)

	in.FilePackages().Add("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageObject(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "go": "latest"
  }
}
-- want --
{
  "packages": {
    "go":     "latest",
    "python": "3.10"
  }
}`)

	in.FilePackages().Add("python@3.10")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageObjectComment(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    // Package comment
    "go": "latest"
  }
}
-- want --
{
  "packages": {
    // Package comment
    "go":     "latest",
    "python": "3.10",
  },
}`)

	in.FilePackages().Add("python@3.10")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageEmptyArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": []
}
-- want --
{
  "packages": ["go@latest"]
}`)

	in.FilePackages().Add("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageOneLineArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": ["go"]
}
-- want --
{
  "packages": [
    "go",
    "python@3.10"
  ]
}`)

	in.FilePackages().Add("python@3.10")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageMultiLineArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": [
    "go"
  ]
}
-- want --
{
  "packages": [
    "go",
    "python@3.10"
  ]
}`)

	in.FilePackages().Add("python@3.10")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPackageArrayComments(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": [
    // Go package comment
    "go",

    // Python package comment
    "python@3.10"
  ]
}
-- want --
{
  "packages": [
    // Go package comment
    "go",

    // Python package comment
    "python@3.10",
    "hello@latest",
  ],
}`)

	in.FilePackages().Add("hello@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestRemovePackageObject(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "go": "latest",
    "python": "3.10"
  }
}
-- want --
{
  "packages": {
    "python": "3.10"
  }
}`)

	in.FilePackages().Remove("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestRemovePackageLastMember(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "env": {"NAME": "value"},
  "packages": {
    "go": "latest"
  }
}
-- want --
{
  "env":      {"NAME": "value"},
  "packages": {}
}`)

	in.FilePackages().Remove("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes(), optBytesToStrings()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestRemovePackageArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": ["go@latest", "python@3.10"]
}
-- want --
{
  "packages": ["python@3.10"]
}`)

	in.FilePackages().Remove("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestRemovePackageLastElement(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": ["go@latest"],
  "env": {
    "NAME": "value"
  }
}
-- want --
{
  "packages": [],
  "env": {
    "NAME": "value"
  }
}`)

	in.FilePackages().Remove("go@latest")
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPlatforms(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "go": {
      "version": "1.20"
    },
    "python": {
      "version": "3.10",
      "platforms": [
        "x86_64-linux"
      ]
    },
    "hello": {
      "version": "latest",
      "platforms": ["x86_64-linux"]
    },
    "vim": {
      "version": "latest"
    }
  }
}
-- want --
{
  "packages": {
    "go": {
      "version":   "1.20",
      "platforms": ["aarch64-darwin", "x86_64-darwin"]
    },
    "python": {
      "version": "3.10",
      "platforms": [
        "x86_64-linux",
        "x86_64-darwin"
      ]
    },
    "hello": {
      "version":   "latest",
      "platforms": ["x86_64-linux", "x86_64-darwin"]
    },
    "vim": {
      "version": "latest"
    }
  }
}`)

	err := in.FilePackages().AddPlatforms(io.Discard, "go@1.20", []string{"aarch64-darwin", "x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.FilePackages().AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.FilePackages().AddPlatforms(io.Discard, "hello@latest", []string{"x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPlatformsMigrateArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": ["go", "python@3.10", "hello"]
}
-- want --
{
  "packages": {
    "go": {
      "platforms": ["aarch64-darwin"]
    },
    "python": {
      "version":   "3.10",
      "platforms": ["x86_64-darwin", "x86_64-linux"]
    },
    "hello": ""
  }
}`)

	err := in.FilePackages().AddPlatforms(io.Discard, "go", []string{"aarch64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.FilePackages().AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin", "x86_64-linux"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestAddPlatformsMigrateArrayComments(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": [
    // Go comment
    "go",

    // Python comment
    "python@3.10"
  ]
}
-- want --
{
  "packages": {
    // Go comment
    "go": "",
    // Python comment
    "python": {
      "version":   "3.10",
      "platforms": ["x86_64-darwin", "x86_64-linux"],
    },
  },
}`)

	err := in.FilePackages().AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin", "x86_64-linux"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestExcludePlatforms(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "go": {
      "version": "1.20"
    }
  }
}
-- want --
{
  "packages": {
    "go": {
      "version":            "1.20",
      "excluded_platforms": ["aarch64-darwin"]
    }
  }
}`)

	err := in.FilePackages().ExcludePlatforms(io.Discard, "go@1.20", []string{"aarch64-darwin"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetOutputs(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "prometheus": {
      "version": "latest"
    }
  }
}
-- want --
{
  "packages": {
    "prometheus": {
      "version": "latest",
      "outputs": ["cli"]
    }
  }
}`)

	err := in.FilePackages().SetOutputs(io.Discard, "prometheus@latest", []string{"cli"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetOutputsMigrateArray(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": ["go", "python@3.10", "prometheus@latest"]
}
-- want --
{
  "packages": {
    "go":     "",
    "python": "3.10",
    "prometheus": {
      "version": "latest",
      "outputs": ["cli"]
    }
  }
}`)

	err := in.FilePackages().SetOutputs(io.Discard, "prometheus@latest", []string{"cli"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetAllowInsecure(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "packages": {
    "python": {
      "version": "2.7"
    }
  }
}
-- want --
{
  "packages": {
    "python": {
      "version":        "2.7",
      "allow_insecure": ["python-2.7.18.1"]
    }
  }
}`)

	err := in.FilePackages().SetAllowInsecure(io.Discard, "python@2.7", []string{"python-2.7.18.1"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestDefault(t *testing.T) {
	path := filepath.Join(t.TempDir())
	in := DefaultConfig()
	inBytes := in.bytes()
	if _, err := hujson.Parse(inBytes); err != nil {
		t.Fatalf("default config JSON is invalid: %v\n%s", err, inBytes)
	}
	err := in.SaveTo(path)
	if err != nil {
		t.Fatal("got save error:", err)
	}
	out, err := Load(filepath.Join(path, defaultName))
	if err != nil {
		t.Fatal("got load error:", err)
	}
	if diff := cmp.Diff(in, out, cmpopts.IgnoreUnexported(configFile{}, packages{})); diff != "" {
		t.Errorf("configs not equal (-in +out):\n%s", diff)
	}

	outBytes := out.bytes()
	if _, err := hujson.Parse(outBytes); err != nil {
		t.Fatalf("loaded default config JSON is invalid: %v\n%s", err, outBytes)
	}
	if string(inBytes) != string(outBytes) {
		t.Errorf("got different JSON after load/save/load:\ninput:\n%s\noutput:\n%s", inBytes, outBytes)
	}
}

func TestNixpkgsValidation(t *testing.T) {
	testCases := map[string]struct {
		commit   string
		isErrant bool
	}{
		"invalid_nixpkg_commit": {"1234545", true},
		"valid_nixpkg_commit":   {"af9e00071d0971eb292fd5abef334e66eda3cb69", false},
	}

	for name, testCase := range testCases {
		t.Run(name, func(t *testing.T) {
			assert := assert.New(t)

			err := ValidateNixpkg(&configFile{
				Nixpkgs: &NixpkgsConfig{
					Commit: testCase.commit,
				},
			})
			if testCase.isErrant {
				assert.Error(err)
			} else {
				assert.NoError(err)
			}
		})
	}
}
