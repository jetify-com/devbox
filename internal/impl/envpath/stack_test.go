package envpath

import (
	"fmt"
	"strings"
	"testing"
)

func TestNewStack(t *testing.T) {
	// Initialize a new Stack from the existing env
	originalEnv := map[string]string{
		"PATH": "/init-path",
	}
	env := make(map[string]string)
	stack := Stack(env, originalEnv)
	if len(stack.keys) == 0 {
		t.Errorf("Stack has no keys but should have %s", InitPathEnv)
	}
	if len(stack.keys) != 1 {
		t.Errorf("Stack has should have exactly one key (%s) but has %d keys. Keys are: %s",
			InitPathEnv, len(stack.keys), strings.Join(stack.keys, ", "))
	}

	// Each testStep below is applied in order, and the resulting env
	// is used implicitly as input into the subsequent test step.
	//
	// These test steps are NOT independent! These are not "test cases" that
	// would usually be independent.
	testSteps := []struct {
		projectHash        string
		devboxEnvPath      string
		preservePathStack  bool
		expectedKeysLength int
		expectedEnv        map[string]string
	}{
		{
			projectHash:        "fooProjectHash",
			devboxEnvPath:      "/foo1:/foo2",
			preservePathStack:  false,
			expectedKeysLength: 2,
			expectedEnv: map[string]string{
				"PATH":                "/foo1:/foo2:/init-path",
				InitPathEnv:           "/init-path",
				Key("fooProjectHash"): "/foo1:/foo2",
			},
		},
		{
			projectHash:        "barProjectHash",
			devboxEnvPath:      "/bar1:/bar2",
			preservePathStack:  false,
			expectedKeysLength: 3,
			expectedEnv: map[string]string{
				"PATH":                "/bar1:/bar2:/foo1:/foo2:/init-path",
				InitPathEnv:           "/init-path",
				Key("fooProjectHash"): "/foo1:/foo2",
				Key("barProjectHash"): "/bar1:/bar2",
			},
		},
		{
			projectHash:        "fooProjectHash",
			devboxEnvPath:      "/foo3:/foo2",
			preservePathStack:  false,
			expectedKeysLength: 3,
			expectedEnv: map[string]string{
				"PATH":                "/foo3:/foo2:/bar1:/bar2:/init-path",
				InitPathEnv:           "/init-path",
				Key("fooProjectHash"): "/foo3:/foo2",
				Key("barProjectHash"): "/bar1:/bar2",
			},
		},
		{
			projectHash:        "barProjectHash",
			devboxEnvPath:      "/bar3:/bar2",
			preservePathStack:  true,
			expectedKeysLength: 3,
			expectedEnv: map[string]string{
				"PATH":                "/foo3:/foo2:/bar3:/bar2:/init-path",
				InitPathEnv:           "/init-path",
				Key("fooProjectHash"): "/foo3:/foo2",
				Key("barProjectHash"): "/bar3:/bar2",
			},
		},
	}

	for idx, testStep := range testSteps {
		t.Run(
			fmt.Sprintf("step_%d", idx), func(t *testing.T) {
				// Push to stack and update PATH env
				stack.Push(env, testStep.projectHash, testStep.devboxEnvPath, testStep.preservePathStack)
				env["PATH"] = stack.Path(env)

				if len(stack.keys) != testStep.expectedKeysLength {
					t.Errorf("Stack should have exactly %d keys but has %d keys. Keys are: %s",
						testStep.expectedKeysLength, len(stack.keys), strings.Join(stack.keys, ", "))
				}
				for k, v := range testStep.expectedEnv {
					if env[k] != v {
						t.Errorf("env[%s] should be %s but is %s", k, v, env[k])
					}
				}
			})
	}
}
