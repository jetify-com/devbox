package configfile

import "go.jetpack.io/devbox/internal/devbox/shellcmd"

type script struct {
	shellcmd.Commands
	Comments string
}

type Scripts map[string]*script

func (c *ConfigFile) Scripts() Scripts {
	if c == nil || c.Shell == nil {
		return nil
	}
	result := make(Scripts)
	for name, commands := range c.Shell.Scripts {
		comments := ""
		if c.ast != nil {
			comments = string(c.ast.beforeComment("shell", "scripts", name))
		}
		result[name] = &script{
			Commands: *commands,
			Comments: comments,
		}
	}

	return result
}
