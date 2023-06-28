package featureflag

// PromptHook controls the insertion of a shell prompt hook that invokes
// devbox shellenv, in lieu of using binary wrappers.
var PromptHook = disable("PROMPT_HOOK")
