package main

import (
	"fmt"
	"runtime"
	"strings"
)

func main() {
	expected := "go1.18"
	goVersion := runtime.Version()
	fmt.Printf("Go version: %s\n", goVersion)
	if !strings.HasPrefix(goVersion, expected) {
		panic(fmt.Errorf("expected version: %s, got: %s", expected, goVersion))
	}
}
