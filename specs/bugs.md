# Known bugs deferred from the bug-fix sweep

These bugs were found during an audit of the devbox codebase. Their fixes were
too large to bundle into the bug-fix PRs, so they are tracked here instead.

## `RunXPaths` races with concurrent devbox commands

**Where:** `internal/devbox/devbox.go`, `RunXPaths` (around line 1198).

**Bug:** Every call to `computeEnv` runs `RunXPaths`, which does
`os.RemoveAll(".devbox/virtenv/runx/bin")` and then recreates the directory and
its symlinks. When two devbox commands run at the same time in the same project
(parallel `devbox run`s, or `devbox shellenv` while a shell is open), one
process can delete the directory while the other is creating symlinks in it.
The second process then fails with `ENOENT`/`EEXIST`. Running shells also lose
their runx binaries for the duration of the rebuild, or permanently if the
rebuild fails.

**Why it isn't a one-liner:** Building into a temp directory and renaming it
into place doesn't work, because `rename(2)` can't replace a non-empty
directory. The fix needs to reconcile the directory in place instead of
recreating it.

**Fix sketch:**

1. Build the desired `name → target` map for all runx binaries first.
2. `os.MkdirAll` the `bin` directory. Don't call `RemoveAll`.
3. For each existing entry, remove it only if it's missing from the map or
   points to a different target.
4. Create the missing symlinks, treating `fs.ErrExist` as success when the
   existing link already points to the right target. This tolerates a
   concurrent process creating it first.

**Estimated size:** one file (`internal/devbox/devbox.go`), about 20–25 lines
changed, plus a unit test that runs the function against a temp project dir
with stale and correct links already present.
