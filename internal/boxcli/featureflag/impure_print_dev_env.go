// Copyright 2024 Jetify Inc. and contributors. All rights reserved.
// Use of this source code is governed by the license in the LICENSE file.

package featureflag

// ImpurePrintDevEnv controls whether the `devbox print-dev-env` command
// will be called with the `--impure` flag.
// Using the `--impure` flag will have two consequences:
//  1. All environment variables will be passed to nix, this will enable
//     the usage of flakes that rely on environment variables.
//  2. It will disable nix caching, making the command slower.
var ImpurePrintDevEnv = disable("IMPURE_PRINT_DEV_ENV")
