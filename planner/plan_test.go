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
			BuildStage: &Stage{
				Command: "plan1",
			},
		},
	}
	actual = MergePlans(plan1, plan2)
	assert.Equal(t, expected, actual)
}
