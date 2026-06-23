// Configures Corepack for the Devbox shell. This is the nodejs plugin's
// init_hook, invoked as: node setup-corepack.mjs
//
// The .mjs extension forces Node to treat this as an ES module regardless of
// the project's package.json "type" field. A plain .js file would be parsed as
// CommonJS or ESM depending on that field, so it would break in one case or the
// other (see issue #2856).
//
// It is a no-op unless DEVBOX_COREPACK_ENABLED is set, in which case it:
//   1. Enables Corepack, installing its package-manager shims into the
//      directory given by DEVBOX_COREPACK_BIN_DIR (which the plugin also puts
//      on PATH via its `env` block, so no PATH export is needed here).
//   2. Activates the package manager pinned in the project's package.json
//      "packageManager" field (pnpm, yarn, npm, ...), unless
//      DEVBOX_DISABLE_NODEJS_PACKAGE_MANAGER_AUTODETECT is set.

import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import path from "node:path";

if (!process.env.DEVBOX_COREPACK_ENABLED) {
  process.exit(0);
}

const corepackBinDir = process.env.DEVBOX_COREPACK_BIN_DIR;
if (!corepackBinDir) {
  process.exit(0);
}

// Enable Corepack, installing the pnpm/yarn/npm shims into corepackBinDir.
// Only attempt package-manager activation if enabling succeeded; if it failed
// for any reason (Corepack missing, offline, etc.) there's nothing to activate.
if (enableCorepack()) {
  // Activate the package manager pinned in package.json's "packageManager"
  // field.
  activatePinnedPackageManager();
}

// enableCorepack runs `corepack enable`. It returns true on success. If the
// `corepack` binary itself is missing it prints an actionable warning and
// returns false, because some Node packages (such as nodejs-slim, and Node.js
// 25+) no longer bundle Corepack (see issue #2791). Other failures (e.g. being
// offline) are ignored so they don't block shell initialization.
function enableCorepack() {
  try {
    execFileSync("corepack", ["enable", "--install-directory", corepackBinDir], {
      stdio: "inherit",
    });
    return true;
  } catch (err) {
    if (err && err.code === "ENOENT") {
      warnCorepackUnavailable();
    }
    return false;
  }
}

function warnCorepackUnavailable() {
  process.stderr.write(
    "[devbox] Warning: DEVBOX_COREPACK_ENABLED is set but the `corepack` " +
      "command was not found. Some Node packages (such as nodejs-slim, and " +
      "Node.js 25+) no longer bundle Corepack. Add the `corepack` package to " +
      "your devbox.json (for example, run `devbox add corepack`) to use " +
      "Yarn or pnpm.\n",
  );
}

function activatePinnedPackageManager() {
  if (process.env.DEVBOX_DISABLE_NODEJS_PACKAGE_MANAGER_AUTODETECT) {
    return;
  }

  const projectRoot = process.env.DEVBOX_PROJECT_ROOT;
  if (!projectRoot) {
    return;
  }

  // Read package.json directly rather than importing it: JSON module import
  // syntax differs across Node versions, whereas readFileSync + JSON.parse
  // works everywhere.
  let packageManager;
  try {
    const pkg = JSON.parse(
      readFileSync(path.join(projectRoot, "package.json"), "utf8"),
    );
    ({ packageManager } = pkg);
  } catch {
    // No package.json (or it is unreadable/invalid) — nothing to autodetect.
    return;
  }

  if (!packageManager) {
    return;
  }

  run("corepack", ["prepare", "--activate", packageManager]);
}

// Run a command, inheriting stdio so Corepack's output is visible. Failures
// must not block shell initialization.
function run(command, args) {
  try {
    execFileSync(command, args, { stdio: "inherit" });
  } catch {
    // Ignore: e.g. Corepack unavailable, or offline during activation.
  }
}
