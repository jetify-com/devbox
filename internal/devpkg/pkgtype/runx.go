package pkgtype

import "strings"

func IsRunX(s string) bool {
	return strings.HasPrefix(s, "runx:")
}
