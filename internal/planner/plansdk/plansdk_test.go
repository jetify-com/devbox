// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
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
		ShellInitHook: []string{"c"},
	}
	plan2 = &ShellPlan{
		Definitions:   []string{"a"},
		ShellInitHook: []string{"a", "b"},
	}
	expected = &ShellPlan{
		DevPackages:   []string{},
		Definitions:   []string{"a"},
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
