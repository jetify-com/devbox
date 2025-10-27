package devbox

import "strings"

func isConnectionError(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "no such host") ||
		strings.Contains(err.Error(), "connection refused")
}
