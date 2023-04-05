package featureflag

// ScriptExitOnError controls whether scripts defined in devbox.json
// and executed via `devbox run` should exit if any command within them errors.
var ScriptExitOnError = disabled("SCRIPT_EXIT_ON_ERROR")
