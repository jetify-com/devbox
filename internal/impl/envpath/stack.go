package envpath

import (
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

const (
	// PathStackEnv stores the string representation of the stack, as a ":" separated list.
	// Each element in the list is also the key to the env-var that stores the
	// nixEnvPath for that devbox-project. Except for the last element, which is InitPathEnv.
	PathStackEnv = "DEVBOX_PATH_STACK"

	// InitPathEnv stores the path prior to any devbox shellenv modifying the environment
	InitPathEnv = "DEVBOX_INIT_PATH"
)

// stack has the following design:
// 1. The stack enables tracking which sub-paths in PATH come from which devbox-project
// 2. It is an ordered-list of keys to env-vars that store nixEnvPath values of devbox-projects.
// 3. The final PATH is reconstructed by concatenating the env-var values of each nixEnvPathKey.
// 5. The stack is stored in its own env-var PathStackEnv, shared by all devbox-projects in this shell.
type stack struct {

	// keys holds the stack elements.
	// Earlier (lower index number) keys get higher priority.
	// This keeps the string representation of the stack aligned with the PATH value.
	keys []string
}

func Stack(env map[string]string) *stack {
	stackEnv, ok := env[PathStackEnv]
	if !ok {
		// if path stack is empty, then push the current PATH, which is the
		// external environment prior to any devbox-shellenv being applied to it.
		stackEnv = InitPathEnv
		env[InitPathEnv] = env["PATH"]
	}
	return &stack{
		keys: strings.Split(stackEnv, ":"),
	}
}

// String is the value of the stack stored in its env-var.
func (s *stack) String() string {
	return strings.Join(s.keys, ":")
}

// Key is the element stored in the stack for a devbox-project. It represents
// a pointer to the nixEnvPath value stored in its own env-var, also using this same
// Key.
func Key(projectHash string) string {
	return "DEVBOX_NIX_ENV_PATH_" + projectHash
}

// PushAndUpdateEnv adds the new nixEnvPath for the devbox-project identified by projectHash.
// The nixEnvPath is pushed to the top of the stack (given highest priority), unless preservePathStack
// is enabled.
//
// It also updated the env by modifying the following env-vars:
// 1. nixEnvPath key
// 2. PathStack
// 3. PATH
//
// Returns the modified env map
func (s *stack) PushAndUpdateEnv(
	env map[string]string,
	projectHash string,
	nixEnvPath string,
	preservePathStack bool,
) map[string]string {
	key := Key(projectHash)

	// Add this nixEnvPath to env
	env[key] = nixEnvPath

	// Common case: ensure this key is at the top of the stack
	if !preservePathStack ||
		// Case preservePathStack == true, usually from bin-wrapper or (in future) shell hook.
		// Add this key only if absent from the stack
		!lo.Contains(s.keys, key) {

		s.keys = lo.Uniq(slices.Insert(s.keys, 0, key))
	}
	env[PathStackEnv] = s.String()

	// Look up the paths-list for each stack element, and join them together to get the final PATH.
	pathLists := lo.Map(s.keys, func(part string, idx int) string { return env[part] })
	env["PATH"] = JoinPathLists(pathLists...)
	return env
}

// Has tests if the stack has the specified key. Refer to the Key function for constructing
// the appropriate key for any devbox-project.
func (s *stack) Has(key string) bool {
	return lo.Contains(s.keys, key)
}
