package nix

import "os"

func init() {
	Default.ExtraArgs = Args{
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", "nix-command flakes fetch-closure",
	}

	// Add GitHub access token if available to avoid rate limiting
	// This is a backup in case the config file isn't picked up properly
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		Default.ExtraArgs = append(Default.ExtraArgs,
			"--option", "access-tokens", "github.com="+token)
	}
}

func appendArgs[E any](args Args, new []E) Args {
	for _, elem := range new {
		args = append(args, elem)
	}
	return args
}

func allowUnfreeEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_UNFREE=1")
}

func allowInsecureEnv(curEnv []string) []string {
	return append(curEnv, "NIXPKGS_ALLOW_INSECURE=1")
}
