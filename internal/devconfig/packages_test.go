package devconfig

import (
	"bytes"
	"encoding/json"
	"reflect"
	"testing"

	"go.jetpack.io/devbox/internal/cuecfg"
)

// TestJsonifyConfigPackages tests the jsonMarshal and jsonUnmarshal of the Config.Packages field
func TestJsonifyConfigPackages(t *testing.T) {
	testCases := []struct {
		name       string
		jsonConfig string
		expected   Packages
	}{
		{
			name:       "empty-list",
			jsonConfig: `{"packages":[]}`,
			expected: Packages{
				jsonKind:   jsonList,
				Collection: []Package{},
			},
		},
		{
			name:       "empty-map",
			jsonConfig: `{"packages":{}}`,
			expected: Packages{
				jsonKind:   jsonMap,
				Collection: []Package{},
			},
		},
		{
			name:       "flat-list",
			jsonConfig: `{"packages":["python","hello@latest","go@1.20"]}`,
			expected: Packages{
				jsonKind:   jsonList,
				Collection: packagesFromLegacyList([]string{"python", "hello@latest", "go@1.20"}),
			},
		},
		{
			name:       "map-with-string-value",
			jsonConfig: `{"packages":{"python":"latest","go":"1.20"}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewVersionOnlyPackage("python", "latest"),
					NewVersionOnlyPackage("go", "1.20"),
				},
			},
		},

		{
			name:       "map-with-struct-value",
			jsonConfig: `{"packages":{"python":{"version":"latest"}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("python", map[string]any{"version": "latest"}),
				},
			},
		},
		{
			name:       "map-with-string-and-struct-values",
			jsonConfig: `{"packages":{"go":"1.20","emacs":"latest","python":{"version":"latest"}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewVersionOnlyPackage("go", "1.20"),
					NewVersionOnlyPackage("emacs", "latest"),
					NewPackage("python", map[string]any{"version": "latest"}),
				},
			},
		},
		{
			name: "map-with-platforms",
			jsonConfig: `{"packages":{"python":{"version":"latest",` +
				`"platforms":["x86_64-darwin","aarch64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("python", map[string]any{
						"version":   "latest",
						"platforms": []string{"x86_64-darwin", "aarch64-linux"},
					}),
				},
			},
		},
		{
			name: "map-with-excluded-platforms",
			jsonConfig: `{"packages":{"python":{"version":"latest",` +
				`"excluded_platforms":["x86_64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("python", map[string]any{
						"version":            "latest",
						"excluded_platforms": []string{"x86_64-linux"},
					}),
				},
			},
		},
		{
			name: "map-with-platforms-and-excluded-platforms",
			jsonConfig: `{"packages":{"python":{"version":"latest",` +
				`"platforms":["x86_64-darwin","aarch64-linux"],` +
				`"excluded_platforms":["x86_64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("python", map[string]any{
						"version":            "latest",
						"platforms":          []string{"x86_64-darwin", "aarch64-linux"},
						"excluded_platforms": []string{"x86_64-linux"},
					}),
				},
			},
		},
		{
			name: "map-with-platforms-and-excluded-platforms-local-flake",
			jsonConfig: `{"packages":{"path:my-php-flake#hello":{"version":"latest",` +
				`"platforms":["x86_64-darwin","aarch64-linux"],` +
				`"excluded_platforms":["x86_64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("path:my-php-flake#hello", map[string]any{
						"version":            "latest",
						"platforms":          []string{"x86_64-darwin", "aarch64-linux"},
						"excluded_platforms": []string{"x86_64-linux"},
					}),
				},
			},
		},
		{
			name: "map-with-platforms-and-excluded-platforms-remote-flake",
			jsonConfig: `{"packages":{"github:F1bonacc1/process-compose/v0.43.1":` +
				`{"version":"latest",` +
				`"platforms":["x86_64-darwin","aarch64-linux"],` +
				`"excluded_platforms":["x86_64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("github:F1bonacc1/process-compose/v0.43.1", map[string]any{
						"version":            "latest",
						"platforms":          []string{"x86_64-darwin", "aarch64-linux"},
						"excluded_platforms": []string{"x86_64-linux"},
					}),
				},
			},
		},
		{
			name: "map-with-platforms-and-excluded-platforms-nixpkgs-reference",
			jsonConfig: `{"packages":{"github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello":` +
				`{"version":"latest",` +
				`"platforms":["x86_64-darwin","aarch64-linux"],` +
				`"excluded_platforms":["x86_64-linux"]}}}`,
			expected: Packages{
				jsonKind: jsonMap,
				Collection: []Package{
					NewPackage("github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello", map[string]any{
						"version":            "latest",
						"platforms":          []string{"x86_64-darwin", "aarch64-linux"},
						"excluded_platforms": []string{"x86_64-linux"},
					}),
				},
			},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			config := &Config{}
			if err := json.Unmarshal([]byte(testCase.jsonConfig), config); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if !reflect.DeepEqual(config.Packages, testCase.expected) {
				t.Errorf("expected: %v, got: %v", testCase.expected, config.Packages)
			}

			marshalled, err := json.Marshal(config)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(marshalled) != testCase.jsonConfig {
				t.Errorf("expected: %v, got: %v", testCase.jsonConfig, string(marshalled))
			}

			// We also test cuecfg.Marshal because elsewhere in our code we rely on it.
			// While in this PR it is now a simple wrapper over json.Marshal, we want to
			// ensure that any future changes to that function don't break our code.
			marshalled, err = cuecfg.Marshal(config, ".json")
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			// We need to pretty-print the expected output because cuecfg.Marshal returns
			// the json pretty-printed.
			expected := &bytes.Buffer{}
			if err := json.Indent(expected, []byte(testCase.jsonConfig), "", cuecfg.Indent); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			if string(marshalled) != expected.String() {
				t.Errorf("expected: %v, got: %v", testCase.jsonConfig, string(marshalled))
			}
		})
	}
}

func TestParseVersionedName(t *testing.T) {
	testCases := []struct {
		name            string
		input           string
		expectedName    string
		expectedVersion string
	}{
		{
			name:            "no-version",
			input:           "python",
			expectedName:    "python",
			expectedVersion: "",
		},
		{
			name:            "with-version-latest",
			input:           "python@latest",
			expectedName:    "python",
			expectedVersion: "latest",
		},
		{
			name:            "with-version",
			input:           "python@1.2.3",
			expectedName:    "python",
			expectedVersion: "1.2.3",
		},
		{
			name:            "with-two-@-signs",
			input:           "emacsPackages.@@latest",
			expectedName:    "emacsPackages.@",
			expectedVersion: "latest",
		},
		{
			name:            "with-trailing-@-sign",
			input:           "emacsPackages.@",
			expectedName:    "emacsPackages.@",
			expectedVersion: "",
		},
		{
			name:            "local-flake",
			input:           "path:my-php-flake#hello",
			expectedName:    "path:my-php-flake#hello",
			expectedVersion: "",
		},
		{
			name:            "remote-flake",
			input:           "github:F1bonacc1/process-compose/v0.43.1",
			expectedName:    "github:F1bonacc1/process-compose/v0.43.1",
			expectedVersion: "",
		},
		{
			name:            "nixpkgs-reference",
			input:           "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
			expectedName:    "github:nixos/nixpkgs/5233fd2ba76a3accb5aaa999c00509a11fd0793c#hello",
			expectedVersion: "",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			name, version := parseVersionedName(testCase.input)
			if name != testCase.expectedName {
				t.Errorf("expected: %v, got: %v", testCase.expectedName, name)
			}
			if version != testCase.expectedVersion {
				t.Errorf("expected: %v, got: %v", testCase.expectedVersion, version)
			}
		})
	}
}
