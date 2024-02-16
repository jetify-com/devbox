package devconfig

func (c *configFile) IsEnvsecEnabled() bool {
	// envsec for legacy.
	return c.EnvFromVal == "envsec" || c.EnvFromVal == "jetpack-cloud"
}
