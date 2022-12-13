// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeShellPlans(t *testing.T) {
	plan1 := &ShellPlan{}
	plan2 := &ShellPlan{
		DevPackages:   []string{},
		Definitions:   []string{"a"},
		NixOverlays:   []string{"b"},
		ShellInitHook: []string{"a", "b"},
		GeneratedFiles: map[string]string{
			"a": "b",
		},
	}
	expected := plan2
	actual, err := MergeShellPlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// Test merge array
	plan1 = &ShellPlan{
		Definitions:   []string{"a"},
		NixOverlays:   []string{"b"},
		ShellInitHook: []string{"c"},
	}
	plan2 = &ShellPlan{
		Definitions:   []string{"a"},
		NixOverlays:   []string{"a"},
		ShellInitHook: []string{"a", "b"},
	}
	expected = &ShellPlan{
		DevPackages:   []string{},
		Definitions:   []string{"a"},
		NixOverlays:   []string{"b", "a"},
		ShellInitHook: []string{"c", "a", "b"},
	}
	actual, err = MergeShellPlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// test merging generated files
	plan1 = &ShellPlan{
		GeneratedFiles: map[string]string{
			"a": "b",
			"b": "c",
		},
	}
	plan2 = &ShellPlan{
		GeneratedFiles: map[string]string{
			"a": "b",
			"b": "c",
			"c": "d",
		},
	}
	expected = &ShellPlan{
		DevPackages:   []string{},
		Definitions:   []string{},
		NixOverlays:   []string{},
		ShellInitHook: []string{},
		GeneratedFiles: map[string]string{
			"a": "b",
			"b": "c",
			"c": "d",
		},
	}
	actual, err = MergeShellPlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestMergeUserBuildPlans(t *testing.T) {
	plannerPlan := &BuildPlan{
		DevPackages:     []string{"nodejs"},
		RuntimePackages: []string{"nodejs"},
		InstallStage: &Stage{
			InputFiles: []string{"package.json"},
			Command:    "npm install",
		},
		BuildStage: &Stage{
			InputFiles: []string{"."},
		},
		StartStage: &Stage{
			InputFiles: []string{"."},
			Command:    "npm start",
		},
	}
	cases := []struct {
		name string
		in   *BuildPlan
		out  *BuildPlan
	}{
		{
			name: "empty base plan",
			in:   &BuildPlan{},
			out: &BuildPlan{
				DevPackages:     []string{"nodejs"},
				RuntimePackages: []string{"nodejs"},
				InstallStage: &Stage{
					InputFiles: []string{"package.json"},
					Command:    "npm install",
				},
				BuildStage: &Stage{
					InputFiles: []string{"."},
				},
				StartStage: &Stage{
					InputFiles: []string{"."},
					Command:    "npm start",
				},
			},
		},
		{
			name: "custom commands",
			in: &BuildPlan{
				DevPackages:     []string{"yarn"},
				RuntimePackages: []string{"yarn"},
				InstallStage: &Stage{
					Command: "yarn install",
				},
				BuildStage: &Stage{
					Command: "yarn build",
				},
				StartStage: &Stage{
					Command: "yarn start",
				},
			},
			out: &BuildPlan{
				DevPackages:     []string{"yarn", "nodejs"},
				RuntimePackages: []string{"nodejs"},
				InstallStage: &Stage{
					InputFiles: []string{"package.json"},
					Command:    "yarn install",
				},
				BuildStage: &Stage{
					InputFiles: []string{"."},
					Command:    "yarn build",
				},
				StartStage: &Stage{
					InputFiles: []string{"."},
					Command:    "yarn start",
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := MergeUserBuildPlan(tc.in, plannerPlan)

			assert.NoError(err)
			assert.Equal(tc.out, got, "plans should match")
		})
	}
}
