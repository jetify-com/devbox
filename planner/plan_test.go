// Copyright 2022 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package planner

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergePlans(t *testing.T) {
	// Packages get appended
	plan1 := &BuildPlan{
		Packages: []string{"foo", "bar"},
	}
	plan2 := &BuildPlan{
		Packages: []string{"baz"},
	}
	expected := &BuildPlan{
		Packages: []string{"foo", "bar", "baz"},
	}
	actual := MergePlans(plan1, plan2)
	assert.Equal(t, expected, actual)
}
