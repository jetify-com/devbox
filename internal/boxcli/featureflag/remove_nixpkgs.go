package featureflag

// RemoveNixpkgs will generate flake.nix code that skips downloads of nixpkgs.
// It leverages the search index to directly map <package>@<version> to
// the /nix/store/<hash>-<package>-<version> that can be fetched from
// cache.nixpkgs.org.
var RemoveNixpkgs = enable("REMOVE_NIXPKGS")
