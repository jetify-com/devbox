package boxcli

// Functions that help parse arguments

// If args empty, defaults to the current directory
// Otherwise grabs the path from the first argument
func pathArg(args []string) string {
	if len(args) > 0 {
		return args[0]
	}
	return "."
}
