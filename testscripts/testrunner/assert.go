package testrunner

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"
)

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

// Usage: path.order 'a' 'b' 'c'
// Checks that whatever is in stdout, P, is a string in PATH format (i.e. colon-separated strings), and that
// every one of the arguments ('a', 'b', and 'c') are contained in separate subpaths of P, exactly once, and
// in order.
func assertPathOrder(script *testscript.TestScript, neg bool, args []string) {
	path := script.ReadFile("stdout")
	subpaths := strings.Split(strings.Replace(path, "\n", "", -1), ":")

	allInOrder := containsInOrder(subpaths, args)
	if !neg && !allInOrder {
		script.Fatalf("Did not find all expected in order in subpaths.\n\nSubpaths: %v\nExpected: %v", subpaths, args)
	}
	if neg && allInOrder {
		script.Fatalf("Found all expected in subpaths.\n\nSubpaths: %v\nExpected: %v", subpaths, args)
	}
}

func containsInOrder(subpaths []string, expected []string) bool {
	if len(expected) == 0 {
		return true // no parts passed in, assertion trivially holds.
	}

	if len(subpaths) < len(expected) {
		return false
	}

	i := 0
	j := 0
outer:
	for j < len(expected) {
		currentExpected := expected[j]
		for i < len(subpaths) {
			if strings.Contains(subpaths[i], currentExpected) {
				j++
				i++
				continue outer // found expected, move on to the next expected
			} else {
				i++ // didn't find it, try the next subpath
			}
		}
		return false // ran out of subpaths, but not out of expected, so we fail.
	}

	return true // if we're here, we found everything
}
