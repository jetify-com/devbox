package configfile

import (
	"slices"
	"strings"

	"go.jetify.com/devbox/internal/devbox/shellcmd"
)

type script struct {
	shellcmd.Commands
	Comments string
}

type Scripts map[string]*script

// ScriptWithName pairs a script with its name so callers can iterate over
// scripts in a deterministic order. Scripts are stored in a map, and Go's
// text/template ranges over map keys in sorted (alphabetical) order, which
// doesn't match the order scripts are defined in devbox.json. This type is
// used when generating documentation so the output preserves the user's
// ordering.
type ScriptWithName struct {
	Name     string
	Commands *shellcmd.Commands
	Comments string
}

// ScriptOrder returns the names of the scripts in the order they appear in the
// devbox.json file. Names that can't be determined from the source file (for
// example, when the config wasn't parsed from a file) are omitted; callers
// should treat a missing name as "order unknown".
func (c *ConfigFile) ScriptOrder() []string {
	if c == nil || c.ast == nil {
		return nil
	}
	return c.ast.objectKeysInOrder("shell", "scripts")
}

// InOrder returns the scripts as a slice ordered by the given names. Any
// scripts not present in order (or when order is nil) are appended in
// alphabetical order so the result stays deterministic.
//
// order may contain a name more than once when it's built by concatenating
// multiple sources (e.g. included configs followed by the root config). In
// that case only the last occurrence is used, so a script overridden by a
// later definition appears at that definition's position. This matches the
// merge precedence that produced its value in s (later definitions win).
func (s Scripts) InOrder(order []string) []ScriptWithName {
	result := make([]ScriptWithName, 0, len(s))
	seen := make(map[string]bool, len(s))
	add := func(name string) {
		sc := s[name]
		result = append(result, ScriptWithName{
			Name:     name,
			Commands: &sc.Commands,
			Comments: sc.Comments,
		})
		seen[name] = true
	}

	lastIndex := make(map[string]int, len(order))
	for i, name := range order {
		lastIndex[name] = i
	}
	for i, name := range order {
		if lastIndex[name] != i {
			continue
		}
		if _, ok := s[name]; ok && !seen[name] {
			add(name)
		}
	}

	rest := make([]string, 0, len(s))
	for name := range s {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	slices.Sort(rest)
	for _, name := range rest {
		add(name)
	}

	return result
}

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
