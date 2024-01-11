package devconfig

func (c *Config) IsEnvsecEnabled() bool {
	return c.EnvFrom == "envsec"
}
