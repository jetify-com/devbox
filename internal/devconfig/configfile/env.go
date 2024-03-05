package configfile

func (c *ConfigFile) IsEnvsecEnabled() bool {
	// envsec for legacy.
	return c.EnvFrom == "envsec" || c.EnvFrom == "jetpack-cloud"
}
