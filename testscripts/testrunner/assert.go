package testrunner

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"
)

// Custom assertions we can use inside a testscript.
var assertionMap = map[string]func(ts *testscript.TestScript, neg bool, args []string){
	"env.path.len":  assertPathLength,
	"json.superset": assertJSONSuperset,
}

// Usage: env.path.len <number>
// Checks that the PATH environment variable has the expected number of entries.
func assertPathLength(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		script.Fatalf("usage: env.path.len N")
	}
	expectedN, err := strconv.Atoi(args[0])
	script.Check(err)

	path := script.Getenv("PATH")
	actualN := len(strings.Split(path, ":"))
	if neg {
		if actualN == expectedN {
			script.Fatalf("path length is %d, expected != %d", actualN, expectedN)
		}
	} else {
		if actualN != expectedN {
			script.Fatalf("path length is %d, expected %d", actualN, expectedN)
		}
	}
}

// Usage: json.superset superset.json subset.json
// Checks that the JSON in superset.json contains all the keys and values
// present in subset.json.
func assertJSONSuperset(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		script.Fatalf("usage: json.superset superset.json subset.json")
	}

	if neg {
		script.Fatalf("json.superset does not support negation")
	}

	data1 := script.ReadFile(args[0])
	tree1 := map[string]interface{}{}
	err := json.Unmarshal([]byte(data1), &tree1)
	script.Check(err)

	data2 := script.ReadFile(args[1])
	tree2 := map[string]interface{}{}
	err = json.Unmarshal([]byte(data2), &tree2)
	script.Check(err)

	for expectedKey, expectedValue := range tree2 {
		if actualValue, ok := tree1[expectedKey]; ok {
			if !reflect.DeepEqual(actualValue, expectedValue) {
				script.Fatalf("key '%s': expected '%v', got '%v'", expectedKey, expectedValue, actualValue)
			}
		} else {
			script.Fatalf("key '%s' not found, expected value '%v'", expectedKey, expectedValue)
		}
	}
}
