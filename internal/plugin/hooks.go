package plugin

func InitHooks(pkgs []string, rootDir string) ([]string, error) {
	hooks := []string{}
	for _, pkg := range pkgs {
		c, err := getConfigIfAny(pkg, rootDir)
		if err != nil {
			return nil, err
		}
		if c == nil {
			continue
		}
		hooks = append(hooks, c.Shell.InitHook.Cmds...)
	}
	return hooks, nil
}
