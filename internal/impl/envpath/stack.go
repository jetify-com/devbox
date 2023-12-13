package envpath

import (
	"strings"

	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

const (
	// PathStackEnv stores the string representation of the stack, as a ":" separated list.
	// Each element in the list is also the key to the env-var that stores the
	// devboxEnvPath for that devbox-project. Except for the last element, which is InitPathEnv.
	PathStackEnv = "DEVBOX_PATH_STACK"

	// InitPathEnv stores the path prior to any devbox shellenv modifying the environment
	InitPathEnv = "DEVBOX_INIT_PATH"
)

// stack has the following design:
// 1. The stack enables tracking which sub-paths in PATH come from which devbox-project
// 2. It is an ordered-list of keys to env-vars that store devboxEnvPath values of devbox-projects.
// 3. The final PATH is reconstructed by concatenating the env-var values of each of these keys.
// 4. The stack is stored in its own env-var PathStackEnv, shared by all devbox-projects in this shell.
type stack struct {
	// keys holds the stack elements.
	// Earlier (lower index number) keys get higher priority.
	// This keeps the string representation of the stack aligned with the PATH value.
	keys []string
}

// Stack initializes the path stack in the `env` environment.
// It relies on old state stored in the `originalEnv` environment.
func Stack(env, originalEnv map[string]string) *stack {
	stackEnv, ok := originalEnv[PathStackEnv]
	if !ok || strings.TrimSpace(stackEnv) == "" {
		// if path stack is empty, then push the current PATH, which is the
		// external environment prior to any devbox-shellenv being applied to it.
		stackEnv = InitPathEnv
		env[InitPathEnv] = originalEnv["PATH"]
	}
	return &stack{
		keys: strings.Split(stackEnv, ":"),
	}
}

// String is the value of the stack stored in its env-var.
func (s *stack) String() string {
	return strings.Join(s.keys, ":")
}

func (s *stack) Path(env map[string]string) string {
	// Look up the paths-list for each stack element, and join them together to get the final PATH.
	pathLists := lo.Map(s.keys, func(part string, idx int) string { return env[part] })
	return JoinPathLists(pathLists...)
}

// Key is the element stored in the stack for a devbox-project. It represents
// a pointer to the devboxEnvPath value stored in its own env-var, also using this same Key.
func Key(projectHash string) string {
	return "DEVBOX_NIX_ENV_PATH_" + projectHash
}

// Push adds the new PATH for the devbox-project identified by projectHash.
// This PATH is pushed to the top of the stack (given highest priority),
// unless preservePathStack is enabled.
//
// It also updates the env by modifying the PathStack env-var, and the env-var
// for storing this path.
func (s *stack) Push(
	env map[string]string,
	projectHash string,
	path string, // new PATH of the devbox-project of projectHash
	preservePathStack bool,
) {
	key := Key(projectHash)

	// Add this path to env
	env[key] = path

	// Common case: ensure this key is at the top of the stack
	if !preservePathStack ||
		// Case preservePathStack == true, usually from bin-wrapper or (in future) shell hook.
		// Add this key only if absent from the stack
		!lo.Contains(s.keys, key) {

		s.keys = lo.Uniq(slices.Insert(s.keys, 0, key))
	}
	env[PathStackEnv] = s.String()
}

// Has tests if the stack has the key corresponding to projectHash
func (s *stack) Has(projectHash string) bool {
	return lo.Contains(s.keys, Key(projectHash))
}
