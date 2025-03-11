package testrunner

import (
	"encoding/json"
	"reflect"
	"strconv"
	"strings"

	"github.com/rogpeppe/go-internal/testscript"

	"go.jetify.com/devbox/internal/devconfig"
	"go.jetify.com/devbox/internal/envir"
	"go.jetify.com/devbox/internal/lock"
)

// Usage: env.path.len <number>
// Checks that the PATH environment variable has the expected number of entries.
func assertPathLength(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 1 {
		script.Fatalf("usage: env.path.len N")
	}
	expectedN, err := strconv.Atoi(args[0])
	script.Check(err)

	path := script.Getenv(envir.Path)
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

func assertDevboxJSONPackagesContains(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		script.Fatalf("usage: devboxjson.packages.contains devbox.json value")
	}

	data := script.ReadFile(args[0])
	list := devconfig.Config{}
	err := json.Unmarshal([]byte(data), &list.Root)
	script.Check(err)

	expected := args[1]
	for _, actual := range packagesVersionedNames(list) {
		if actual == expected {
			if neg {
				script.Fatalf("value '%s' found in '%s'", expected, packagesVersionedNames(list))
			}
			return
		}
	}

	if !neg {
		script.Fatalf("value '%s' not found in '%s'", expected, packagesVersionedNames(list))
	}
}

func assertDevboxLockPackagesContains(script *testscript.TestScript, neg bool, args []string) {
	if len(args) != 2 {
		script.Fatalf("usage: devboxlock.packages.contains devbox.lock value")
	}

	data := script.ReadFile(args[0])
	lockfile := lock.File{}
	err := json.Unmarshal([]byte(data), &lockfile)
	script.Check(err)

	expected := args[1]
	if _, ok := lockfile.Packages[expected]; ok {
		if neg {
			script.Fatalf("value '%s' found in %s", expected, args[0])
		}
	} else {
		if !neg {
			script.Fatalf("value '%s' not found in '%s'", expected, args[0])
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
			sortIfPossible(actualValue)
			sortIfPossible(expectedValue)

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

func containsInOrder(subpaths, expected []string) bool {
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

func sortIfPossible(v any) {
	if slice, ok := v.([]any); ok {
		for i := 0; i < len(slice); i++ {
			for j := i + 1; j < len(slice); j++ {
				if compare(slice[i], slice[j]) > 0 {
					slice[i], slice[j] = slice[j], slice[i]
				}
			}
		}
	}
}

func compare(one, two any) int {
	aType, bType := reflect.TypeOf(one), reflect.TypeOf(two)

	if aType.Kind() == bType.Kind() {
		switch aType.Kind() {
		case reflect.Int:
			aInt := one.(int)
			bInt := two.(int)
			return aInt - bInt
		case reflect.String:
			aStr := one.(string)
			bStr := two.(string)
			return strings.Compare(aStr, bStr)
		}
	}

	return 0
}

func packagesVersionedNames(c devconfig.Config) []string {
	result := make([]string, 0, len(c.Root.TopLevelPackages()))
	for _, p := range c.Root.TopLevelPackages() {
		result = append(result, p.VersionedName())
	}
	return result
}
