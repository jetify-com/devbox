// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

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
		SharedPlan: SharedPlan{
			InstallStage: &Stage{
				Command:    "",
				InputFiles: []string{"."},
			},
			BuildStage: &Stage{
				Command: "",
			},
			StartStage: &Stage{
				Command:    "",
				InputFiles: []string{"."},
			},
		},
	}
	actual := MergePlans(plan1, plan2)
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
			InstallStage: &Stage{
				Command:    "",
				InputFiles: []string{"."},
			},
			BuildStage: &Stage{
				Command: "plan1",
			},
			StartStage: &Stage{
				Command:    "",
				InputFiles: []string{"."},
			},
		},
	}
	actual = MergePlans(plan1, plan2)
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
				Command:    "",
				InputFiles: []string{"package.json"},
			},
			BuildStage: &Stage{
				Command: "",
			},
			StartStage: &Stage{
				Command:    "",
				InputFiles: []string{"input"},
			},
		},
	}
	actual = MergePlans(plan1, plan2)
	assert.Equal(t, expected, actual)
}
