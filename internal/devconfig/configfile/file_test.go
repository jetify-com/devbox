//nolint:varnamelen
package configfile

import (
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func parseConfigTxtarTest(t *testing.T, test string) (in *ConfigFile, want []byte) {
	t.Helper()

	ar := txtar.Parse([]byte(test))
	if comment := strings.TrimSpace(string(ar.Comment)); comment != "" {
		t.Log(comment)
	}
	for _, f := range ar.Files {
		switch f.Name {
		case "in":
			var err error
			in, err = LoadBytes(f.Data)
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

	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("python@3.10")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("python@3.10")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("python@3.10")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("python@3.10")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Add("hello@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Remove("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Remove("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes(), optBytesToStrings()); diff != "" {
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

	in.PackagesMutator.Remove("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	in.PackagesMutator.Remove("go@latest")
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.AddPlatforms(io.Discard, "go@1.20", []string{"aarch64-darwin", "x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.PackagesMutator.AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.PackagesMutator.AddPlatforms(io.Discard, "hello@latest", []string{"x86_64-darwin"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.AddPlatforms(io.Discard, "go", []string{"aarch64-darwin"})
	if err != nil {
		t.Error(err)
	}
	err = in.PackagesMutator.AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin", "x86_64-linux"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.AddPlatforms(io.Discard, "python@3.10", []string{"x86_64-darwin", "x86_64-linux"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.ExcludePlatforms(io.Discard, "go@1.20", []string{"aarch64-darwin"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.SetOutputs(io.Discard, "prometheus@latest", []string{"cli"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.SetOutputs(io.Discard, "prometheus@latest", []string{"cli"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
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

	err := in.PackagesMutator.SetAllowInsecure(io.Discard, "python@2.7", []string{"python-2.7.18.1"})
	if err != nil {
		t.Error(err)
	}
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetEnv(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{}
-- want --
{
  "env": {
    "BAZ": "qux",
    "FOO": "bar"
  }
}`)

	in.SetEnv(map[string]string{
		"FOO": "bar",
		"BAZ": "qux",
	})
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetEnvExisting(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "env": {
    "EXISTING": "value"
  }
}
-- want --
{
  "env": {
    "FOO": "bar"
  }
}`)

	in.SetEnv(map[string]string{
		"FOO": "bar",
	})
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
	}
}

func TestSetEnvClear(t *testing.T) {
	in, want := parseConfigTxtarTest(t, `
-- in --
{
  "env": {
    "EXISTING": "value"
  }
}
-- want --
{
  "env": {}
}`)

	in.SetEnv(map[string]string{})
	if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
		t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
	}
	if diff := cmp.Diff(want, in.Bytes()); diff != "" {
		t.Errorf("wrong raw config hujson (-want +got):\n%s", diff)
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

			err := ValidateNixpkg(&ConfigFile{
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

func TestExcludeCCToolchain(t *testing.T) {
	testCases := map[string]struct {
		config *ConfigFile
		want   bool
	}{
		"nil_config": {
			config: nil,
			want:   false,
		},
		"nil_shell": {
			config: &ConfigFile{},
			want:   false,
		},
		"false_value": {
			config: &ConfigFile{
				Shell: &shellConfig{
					ExcludeCCToolchain: false,
				},
			},
			want: false,
		},
		"true_value": {
			config: &ConfigFile{
				Shell: &shellConfig{
					ExcludeCCToolchain: true,
				},
			},
			want: true,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			got := tc.config.ExcludeCCToolchain()
			if got != tc.want {
				t.Errorf("ExcludeCCToolchain() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestExcludeCCToolchainParsing(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name: "explicit_true",
			input: `
-- in --
{
  "packages": ["nodejs@18"],
  "shell": {
    "exclude_cc_toolchain": true
  }
}
-- want --
{
  "packages": ["nodejs@18"],
  "shell": {
    "exclude_cc_toolchain": true
  }
}`,
			expected: true,
		},
		{
			name: "explicit_false",
			input: `
-- in --
{
  "packages": ["python@3.11"],
  "shell": {
    "exclude_cc_toolchain": false
  }
}
-- want --
{
  "packages": ["python@3.11"],
  "shell": {
    "exclude_cc_toolchain": false
  }
}`,
			expected: false,
		},
		{
			name: "omitted_defaults_to_false",
			input: `
-- in --
{
  "packages": ["go@1.20"]
}
-- want --
{
  "packages": ["go@1.20"]
}`,
			expected: false,
		},
		{
			name: "with_other_shell_config",
			input: `
-- in --
{
  "packages": ["ruby@3.2"],
  "shell": {
    "init_hook": ["echo 'hello'"],
    "exclude_cc_toolchain": true,
    "scripts": {
      "test": "echo 'test'"
    }
  }
}
-- want --
{
  "packages": ["ruby@3.2"],
  "shell": {
    "init_hook": ["echo 'hello'"],
    "exclude_cc_toolchain": true,
    "scripts": {
      "test": "echo 'test'"
    }
  }
}`,
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			in, want := parseConfigTxtarTest(t, tc.input)

			got := in.ExcludeCCToolchain()
			if got != tc.expected {
				t.Errorf("ExcludeCCToolchain() = %v, want %v", got, tc.expected)
			}

			if diff := cmp.Diff(want, in.Bytes(), optParseHujson()); diff != "" {
				t.Errorf("wrong parsed config json (-want +got):\n%s", diff)
			}
		})
	}
}
