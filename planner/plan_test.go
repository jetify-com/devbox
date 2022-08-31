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
		Packages: []string{"foo", "bar"},
	}
	plan2 := &Plan{
		Packages: []string{"baz"},
	}
	expected := &Plan{
		Packages: []string{"foo", "bar", "baz"},
	}
	actual := MergePlans(plan1, plan2)
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
		Packages: []string{},
		BuildStage: &Stage{
			Command: "plan1",
		},
	}
	actual = MergePlans(plan1, plan2)
	assert.Equal(t, expected, actual)
}
