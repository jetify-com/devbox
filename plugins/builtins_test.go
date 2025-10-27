package plugins

import (
	"testing"
)

func TestBuiltInMap(t *testing.T) {
	testCases := map[string]string{
		"apache":                                 "apacheHttpd",
		"apacheHttpd":                            "apacheHttpd",
		"php":                                    "php",
		"php81":                                  "php",
		"php82":                                  "php",
		"gradle":                                 "gradle",
		"gradle_7":                               "gradle",
		"ghc":                                    "haskell",
		"haskell.compiler.abc":                   "haskell",
		"haskell.compiler.native-bignum.ghcHEAD": "haskell",
		"haskell.compiler.native-bignum.ghc962":  "haskell",
		"mariadb":                                "mariadb",
		"mariadb_1011":                           "mariadb",
		"mariadb-embedded":                       "mariadb",
		"mysql":                                  "mariadb",
		"mysql80":                                "mysql",
		"python3Packages.pip":                    "pip",
		"python":                                 "python",
		"python3":                                "python",
		"python3Full":                            "python",
		"python2Minimal":                         "python",
		"python-full":                            "python",
		"python-minimal":                         "python",
		"redis":                                  "redis",
		"ruby":                                   "ruby",
		"ruby_21":                                "ruby",
		"ruby_2_6":                               "ruby",
		"ruby_2_6_5":                             "ruby",
		"ruby_2_6_5_1":                           "ruby",
		"ruby_2_6_5_1_2":                         "ruby",
		"jruby":                                  "ruby",
		"ruby_":                                  "",
		"ruby_abc":                               "",
		"ruby_2_":                                "",
	}

	for input, expected := range testCases {
		matched := false
		for re, value := range builtInMap {
			if re.MatchString(input) {
				matched = true
				if value != expected {
					t.Errorf("Regex match failed for input: %s. Expected: %s, Got: %s", input, expected, value)
				}
			}
		}
		if !matched && expected != "" {
			t.Errorf("Regex match failed for input: %s. Expected: %s, Got: %s", input, expected, "")
		}
	}
}
