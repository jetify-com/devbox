package bincache

import "os"

// ExtraSubstituter returns the URL of the extra substituter to use.
// a substituter is a bin cache url that nix can use to fetch pre-built
// binaries from.
func ExtraSubstituter() (string, error) {
	if err := ensureTrustedUser(); err != nil {
		return "", err
	}

	// TODO: if user is logged in (or if we have token we can refresh)
	// then we try to fetch the bincache URL from the API.

	// DEVBOX_NIX_BINCACHE_URL seems like a friendlier name than "substituter"
	return os.Getenv("DEVBOX_NIX_BINCACHE_URL"), nil
}

func ensureTrustedUser() error {
	// TODO: we need to ensure that the user can actually use the extra
	// substituter. If the user did a root install, then we need to add
	// the extra substituter to the nix.conf file and restart the daemon.
	return nil
}
