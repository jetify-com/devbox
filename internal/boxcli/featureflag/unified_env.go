// Copyright 2023 Jetpack Technologies Inc and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

// UnifiedEnv controls the implementation of `devbox run` and `devbox shell`.
// When enabled, these commands are executed by spawning a shell and passing it
// an environment that was computed primarily based on `nix print-dev-env`. The
// modifications that we make to the output of `nix print-dev-env` are documented
// in impl.computeNixEnv().
// The feature is called UnifiedEnv because we use the exact same environment for
// both devbox shell and devbox run.
var UnifiedEnv = enabled("UNIFIED_ENV")
