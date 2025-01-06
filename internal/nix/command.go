package nix

func init() {
	Default.ExtraArgs = Args{
		"--extra-experimental-features", "ca-derivations",
		"--option", "experimental-features", "nix-command flakes fetch-closure",
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
