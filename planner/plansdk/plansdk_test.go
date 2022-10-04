// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package plansdk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergePlans(t *testing.T) {
	// Packages get appended
	plan1 := &Plan{
		DevPackages:     []string{"foo", "bar"},
		RuntimePackages: []string{"a"},
	}
	plan2 := &Plan{
		DevPackages:     []string{"baz"},
		RuntimePackages: []string{"b", "c"},
	}
	expected := &Plan{
		NixOverlays:     []string{},
		DevPackages:     []string{"foo", "bar", "baz"},
		RuntimePackages: []string{"a", "b", "c"},
	}
	actual, err := MergePlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// Base plan (the first one) takes precedence:
	plan1 = &Plan{
		BuildStage: &Stage{
			Command: "plan1",
		},
	}
	plan2 = &Plan{
		BuildStage: &Stage{
			Command: "plan2",
		},
	}
	expected = &Plan{
		NixOverlays:     []string{},
		DevPackages:     []string{},
		RuntimePackages: []string{},
		BuildStage: &Stage{
			Command: "plan1",
		},
	}
	actual, err = MergePlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// InputFiles can be overwritten:
	plan1 = &Plan{
		InstallStage: &Stage{
			InputFiles: []string{"package.json"},
		},
		StartStage: &Stage{
			InputFiles: []string{"input"},
		},
	}
	plan2 = &Plan{}
	expected = &Plan{
		NixOverlays:     []string{},
		DevPackages:     []string{},
		RuntimePackages: []string{},
		InstallStage: &Stage{
			InputFiles: []string{"package.json"},
		},
		StartStage: &Stage{
			InputFiles: []string{"input"},
		},
	}
	actual, err = MergePlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)
}

func TestMergeUserPlans(t *testing.T) {
	plannerPlan := &Plan{
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
		in   *Plan
		out  *Plan
	}{
		{
			name: "empty base plan",
			in:   &Plan{},
			out: &Plan{
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
				NixOverlays: []string{},
			},
		},
		{
			name: "custom commands",
			in: &Plan{
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
			out: &Plan{
				DevPackages:     []string{"yarn", "nodejs"},
				RuntimePackages: []string{"yarn", "nodejs"},
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
				NixOverlays: []string{},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert := assert.New(t)
			got, err := MergeUserPlan(tc.in, plannerPlan)

			assert.NoError(err)
			assert.Equal(tc.out, got, "plans should match")
		})
	}
}
