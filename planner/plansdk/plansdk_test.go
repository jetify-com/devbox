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
		DevPackages:     []string{"foo", "bar", "baz"},
		RuntimePackages: []string{"a", "b", "c"},
		SharedPlan:      SharedPlan{},
	}
	actual, err := MergePlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// Base plan (the first one) takes precedence:
	plan1 = &Plan{
		SharedPlan: SharedPlan{
			BuildStage: &Stage{
				Command: "plan1",
			},
		},
	}
	plan2 = &Plan{
		SharedPlan: SharedPlan{
			BuildStage: &Stage{
				Command: "plan2",
			},
		},
	}
	expected = &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
		SharedPlan: SharedPlan{
			BuildStage: &Stage{
				Command: "plan1",
			},
		},
	}
	actual, err = MergePlans(plan1, plan2)
	assert.NoError(t, err)
	assert.Equal(t, expected, actual)

	// InputFiles can be overwritten:
	plan1 = &Plan{
		SharedPlan: SharedPlan{
			InstallStage: &Stage{
				InputFiles: []string{"package.json"},
			},
			StartStage: &Stage{
				InputFiles: []string{"input"},
			},
		},
	}
	plan2 = &Plan{
		SharedPlan: SharedPlan{},
	}
	expected = &Plan{
		DevPackages:     []string{},
		RuntimePackages: []string{},
		SharedPlan: SharedPlan{
			InstallStage: &Stage{
				InputFiles: []string{"package.json"},
			},
			StartStage: &Stage{
				InputFiles: []string{"input"},
			},
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
		SharedPlan: SharedPlan{
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
	}
	cases := []struct {
		name string
		in   *Plan
		out  *Plan
	}{
		{
			name: "empty base plan",
			in: &Plan{
				SharedPlan: SharedPlan{},
			},
			out: &Plan{
				DevPackages:     []string{"nodejs"},
				RuntimePackages: []string{"nodejs"},
				SharedPlan: SharedPlan{
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
		},
		{
			name: "different input files",
			in: &Plan{
				DevPackages:     []string{"nodejs", "yarn"},
				RuntimePackages: []string{"nodejs"},
				SharedPlan: SharedPlan{
					InstallStage: &Stage{
						InputFiles: []string{"package.json", "yarn.lock"},
						Command:    "",
					},
					BuildStage: &Stage{
						InputFiles: []string{"."},
						Command:    "",
					},
					StartStage: &Stage{
						InputFiles: []string{"."},
						Command:    "npm start",
					},
				},
			},
			out: &Plan{
				DevPackages:     []string{"nodejs", "yarn"},
				RuntimePackages: []string{"nodejs"},
				SharedPlan: SharedPlan{
					InstallStage: &Stage{
						InputFiles: []string{"package.json", "yarn.lock"},
						Command:    "npm install",
					},
					BuildStage: &Stage{
						InputFiles: []string{"."},
						Command:    "",
					},
					StartStage: &Stage{
						InputFiles: []string{"."},
						Command:    "npm start",
					},
				},
			},
		},
		{
			name: "custom build command",
			in: &Plan{
				SharedPlan: SharedPlan{
					InstallStage: &Stage{
						InputFiles: []string{"app"},
					},
					BuildStage: &Stage{
						Command: "npm run build",
					},
				},
			},
			out: &Plan{
				DevPackages:     []string{"nodejs"},
				RuntimePackages: []string{"nodejs"},
				SharedPlan: SharedPlan{
					InstallStage: &Stage{
						InputFiles: []string{"app"},
						Command:    "npm install",
					},
					BuildStage: &Stage{
						InputFiles: []string{"."},
						Command:    "npm run build",
					},
					StartStage: &Stage{
						InputFiles: []string{"."},
						Command:    "npm start",
					},
				},
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
