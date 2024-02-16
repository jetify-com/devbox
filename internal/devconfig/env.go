package devconfig

func (c *configFile) IsEnvsecEnabled() bool {
	// envsec for legacy.
	return c.EnvFrom == "envsec" || c.EnvFrom == "jetpack-cloud"
}
