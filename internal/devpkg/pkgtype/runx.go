package pkgtype

import "strings"

const RunXScheme = "runx"
const RunXPrefix = RunXScheme + ":"

func IsRunX(s string) bool {
	return strings.HasPrefix(s, RunXPrefix)
}
