package testrunner

import (
	"os"
	"testing"

	"go.jetpack.io/devbox/testscripts/testrunner"
)

func TestScripts(t *testing.T) {
	testrunner.RunTestscripts(t, ".")
}

func TestMain(m *testing.M) {
	os.Exit(testrunner.Main(m))
}
