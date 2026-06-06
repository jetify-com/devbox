// Configures Corepack for the Devbox shell. This is the nodejs plugin's
// init_hook, invoked as:
//
//   eval "$(node setup-corepack.js <corepack-bin-dir>)"
//
// It is a no-op unless DEVBOX_COREPACK_ENABLED is set, in which case it:
//   1. Enables Corepack, installing its package-manager shims into
//      <corepack-bin-dir>.
//   2. Activates the package manager pinned in the project's package.json
//      "packageManager" field (pnpm, yarn, npm, ...), unless
//      DEVBOX_DISABLE_NODEJS_PACKAGE_MANAGER_AUTODETECT is set.
//   3. Prints an `export PATH=...` line to stdout so the calling shell, via
//      `eval`, puts those shims on PATH.
//
// IMPORTANT: stdout is consumed by `eval`, so only the final `export` line may
// be written to it. Everything else (including child-process output) goes to
// stderr.

const { execFileSync } = require("node:child_process");
const path = require("node:path");

if (!process.env.DEVBOX_COREPACK_ENABLED) {
  process.exit(0);
}

const corepackBinDir = process.argv[2];
if (!corepackBinDir) {
  process.exit(0);
}

// Enable Corepack, installing the pnpm/yarn/npm shims into corepackBinDir.
run("corepack", ["enable", "--install-directory", corepackBinDir]);

// Activate the package manager pinned in package.json's "packageManager" field.
activatePinnedPackageManager();

// Emit the PATH update for the calling shell to `eval` so the shims are on PATH.
process.stdout.write(`export PATH="${corepackBinDir}:$PATH"\n`);

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

  run("corepack", ["prepare", "--activate", packageManager]);
}

// Run a command, routing its output to stderr (fd 2) so it never pollutes the
// stdout that the shell will `eval`. Failures must not block shell init.
function run(command, args) {
  try {
    execFileSync(command, args, { stdio: ["ignore", 2, 2] });
  } catch {
    // Ignore: e.g. Corepack unavailable, or offline during activation.
  }
}
