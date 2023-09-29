package envpath

import (
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

const (
	PathStackEnv = "DEVBOX_PATH_STACK"

	// InitPathEnv stores the path prior to any devbox shellenv modifying the environment
	InitPathEnv = "DEVBOX_INIT_PATH"
)

// Stack has the following design:
// 1. The PathStack enables tracking which sub-paths in PATH come from which devbox-project
// 2. What it stores: The PathStack is an ordered-list of nixEnvPathKeys
// 3. Each nixEnvPathKey is set as an env-var with the value of the nixEnvPath for that devbox-project.
// 4. The final PATH is reconstructed by concatenating the env-var values of each nixEnvPathKey env-var.
// 5. The Stack is stored in its own env-var shared by all devbox-projects in this shell.
type Stack struct {

	// keys holds the stack elements.
	// Earlier (lower index number) keys get higher priority.
	// This keeps the string representation of the Stack aligned with the PATH value.
	keys []string
}

func NewStack(env map[string]string) *Stack {
	stackEnv, ok := env[PathStackEnv]
	if !ok {
		// if path stack is empty, then push the current PATH, which is the
		// external environment prior to any devbox-shellenv being applied to it.
		stackEnv = InitPathEnv
		env[InitPathEnv] = env["PATH"]
	}
	return &Stack{
		keys: strings.Split(stackEnv, ":"),
	}
}

// String is the value of the Stack stored in its env-var.
func (s *Stack) String() string {
	return strings.Join(s.keys, ":")
}

// Key is the element stored in the Stack for a devbox-project. It represents
// a pointer to the nixEnvPath value stored in its own env-var, also using this same
// Key.
func Key(projectHash string) string {
	return "DEVBOX_NIX_" + projectHash
}

// AddToEnv adds the new nixEnvPath for the devbox-project identified by projectHash to the env.
// It does so by modifying the following env-vars:
// 1. nixEnvPath key
// 2. PathStack
// 3. PATH
//
// Returns the modified env map
func (s *Stack) AddToEnv(
	env map[string]string,
	projectHash string,
	nixEnvPath string,
	pathStackInPlace bool,
) map[string]string {
	key := Key(projectHash)

	// Add this nixEnvPath to env
	env[key] = nixEnvPath

	// Common case: ensure this key is at the top of the stack
	if !pathStackInPlace ||
		// Case pathStackInPlace == true, usually from bin-wrapper or (in future) shell hook.
		// Add this key only if absent from the stack
		!lo.Contains(s.keys, key) {

		s.keys = lo.Uniq(slices.Insert(s.keys, 0, key))
	}
	env[PathStackEnv] = s.String()

	// Look up the paths-list for each paths-stack element, and join them together to get the final PATH.
	pathLists := lo.Map(s.keys, func(part string, idx int) string { return env[part] })
	env["PATH"] = JoinPathLists(pathLists...)
	return env
}

// Has tests if the stack has the specified key. Refer to the Key function for constructing
// the appropriate key for any devbox-project.
func (s *Stack) Has(key string) bool {
	for _, k := range s.keys {
		if k == key {
			return true
		}
	}
	return false
}
