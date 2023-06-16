package featureflag

// PromptHook controls the insertion of a shell prompt hook that invokes
// devbox shellenv, in lieu of using binary wrappers.
var PromptHook = disabled("PROMPT_HOOK")
