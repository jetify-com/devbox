package configfile

import (
	"strings"

	"go.jetify.com/devbox/internal/devbox/shellcmd"
)

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

func (s Scripts) WithRelativePaths(projectDir string) Scripts {
	result := make(Scripts, len(s))
	for name, s := range s {
		commandsWithRelativePaths := shellcmd.Commands{}
		for _, c := range s.Commands.Cmds {
			commandsWithRelativePaths.Cmds = append(
				commandsWithRelativePaths.Cmds,
				strings.ReplaceAll(c, projectDir, "."),
			)
		}
		result[name] = &script{
			Commands: commandsWithRelativePaths,
			Comments: s.Comments,
		}
	}
	return result
}
