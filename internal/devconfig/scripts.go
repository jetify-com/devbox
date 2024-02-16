package devconfig

import "go.jetpack.io/devbox/internal/devbox/shellcmd"

type script struct {
	shellcmd.Commands
	Comments string
}

type scripts map[string]*script

func (c *configFile) Scripts() scripts {
	if c == nil || c.Shell == nil {
		return nil
	}
	result := make(scripts)
	for name, commands := range c.Shell.Scripts {
		result[name] = &script{
			Commands: *commands,
			Comments: string(c.ast.beforeComment("shell", "scripts", name)),
		}
	}

	return result
}
