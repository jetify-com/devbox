// Activates the package manager pinned in the project's package.json
// "packageManager" field via Corepack, so the pinned version (pnpm, yarn, npm,
// ...) is available in the Devbox shell.
//
// This script is invoked by the nodejs plugin's init_hook only when Corepack is
// enabled (DEVBOX_COREPACK_ENABLED). Set
// DEVBOX_DISABLE_NODEJS_PACKAGE_MANAGER_AUTODETECT=1 to disable it.

const { execFileSync } = require("node:child_process");
const path = require("node:path");

function activatePinnedPackageManager() {
  if (process.env.DEVBOX_DISABLE_NODEJS_PACKAGE_MANAGER_AUTODETECT) {
    return;
  }

  const projectRoot = process.env.DEVBOX_PROJECT_ROOT;
  if (!projectRoot) {
    return;
  }

  let packageManager;
  try {
    ({ packageManager } = require(path.join(projectRoot, "package.json")));
  } catch {
    // No package.json (or it is unreadable) — nothing to autodetect.
    return;
  }

  if (!packageManager) {
    return;
  }

  try {
    execFileSync("corepack", ["prepare", "--activate", packageManager], {
      stdio: "inherit",
    });
  } catch {
    // Don't block shell initialization if activation fails (e.g. offline).
  }
}

activatePinnedPackageManager();
