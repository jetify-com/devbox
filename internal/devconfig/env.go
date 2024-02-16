package devconfig

func (c *Config) IsEnvsecEnabled() bool {
	// envsec for legacy.
	return c.EnvFrom == "envsec" || c.EnvFrom == "jetpack-cloud"
}
