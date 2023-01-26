Test devbox using the testscripts framework.

This directory contains testscripts: files ending in `.test.txt` that we
automatically run using the testscripts framework.

For details on how to write these types of files see:
+ https://bitfieldconsulting.com/golang/test-scripts
+ https://pkg.go.dev/github.com/rogpeppe/go-internal/testscript

In addition to the standard testscript commands, we've also support running devbox
commands. Examples include:
+ `devbox init`
+ `devbox add <pkg>`
+ ...
