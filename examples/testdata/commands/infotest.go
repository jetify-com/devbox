package main

import (
	"fmt"

	"go.jetpack.io/devbox/examples/testdata/testframework"
)

func main() {
	td := testframework.Open()
	output, _ := td.Info("sdfagdsg", false)

	fmt.Println(output + "testsetsts")
}
