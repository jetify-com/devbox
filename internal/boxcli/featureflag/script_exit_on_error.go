// Copyright 2024 Jetify Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

// ScriptExitOnError controls whether scripts defined in devbox.json
// and executed via `devbox run` should exit if any command within them errors.
var ScriptExitOnError = enable("SCRIPT_EXIT_ON_ERROR")
