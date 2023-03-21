package main

import (
	"fmt"
	"runtime"
)

func main() {
	expected := "go1.19.4"
	goVersion := runtime.Version()
	fmt.Printf("Go version: %s\n", goVersion)
	if goVersion != expected {
		panic(fmt.Errorf("expected version: %s, got: %s", expected, goVersion))
	}
}
