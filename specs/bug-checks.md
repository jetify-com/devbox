# Devbox bug checks

Results of an audit of the devbox Go code (~42k lines). The audit had two parts:

1. **Automated checks:** `go build`, `go vet`, `golangci-lint`, and `go test -race ./...`.
2. **Manual review:** reviewers read the code subsystem by subsystem, and each finding was checked against the surrounding code.

Severity is **High / Medium / Low**. Line numbers refer to the commit this file was added in.

## 1. Automated checks

| Check | Result |
|---|---|
| `go build ./...` | Passes |
| `go vet ./...` | Clean, no findings |
| `golangci-lint run` | **Could not run.** The installed binary was built with go1.25, but `go.mod` targets go1.27, so it refused to load the config. CI needs a golangci-lint built with go ≥ 1.27, or lint is silently skipped (see T1). |
| `go test -race ./...` | No data races were reported. Failures are listed below. |

### Test failures

Most failures came from the audit sandbox: it runs as root and its network proxy blocks `api.github.com`. Three failures point at real issues and are listed first.

| Test | Cause | Real bug? |
|---|---|---|
| `testscripts/*` (~35 scripts), `TestShellPath/pure_mode_enabled` | `nix print-dev-env` / `path-info` got HTTP 403 from `api.github.com` | Environment. However, see T2: devbox reports this network error as *"There was an internal error … consider reporting it"*. |
| `TestDevbox/{github,github-with-revision,v2-github}_shell_plan` | 404 fetching `raw.githubusercontent.com/jetify-com/devbox-plugin-example/...` | Environment (proxy token). However, see T3: the unit test depends on the live network. |
| `devconfig.TestFindError/{Permissions,ExactFilePermissions}` | Running as root bypasses `chmod 000` | Environment. However, see T4. |
| `nix.TestConfigIsUserTrusted/NotInList` | The test user is `root`, and `trusted-users = root` | Environment. However, see T4. |

### Test and tooling issues

- **T1 (Low): golangci-lint version drift.** The lint config can only be loaded by a golangci-lint built with Go ≥ 1.27. Pin the lint version in `devbox.json` or CI, so that a stale local binary fails loudly instead of skipping lint.
- **T2 (Low): network errors are labelled "internal error".**
  - **Problem:** A 403 or offline failure while nix fetches nixpkgs is wrapped as an internal error with a "please report this bug" message.
  - **Fix:** Nix fetch and download failures (`unable to download`, `HTTP error`) should be classified as user or environment errors.
- **T3 (Low): `internal/devbox/devbox_test.go` depends on the network.** The `github*` example plugins are fetched from GitHub, so the unit tests fail offline or behind a proxy. Mock the plugin fetch, or skip these tests when the network is unavailable.
- **T4 (Low): tests assume a non-root user.** `TestFindError` permission cases and `TestConfigIsUserTrusted/NotInList` fail when run as root, which is common in containers and CI. Skip them when `os.Geteuid() == 0`, or use a username that cannot be trusted.

## 2. Code review findings

### Summary

| # | Severity | Area | Bug |
|---|---|---|---|
| 1 | **High** | nix | `GITHUB_TOKEN` leaks into nix error messages, debug logs and telemetry |
| 2 | **High** | shell | `SHELL=/bin/sh` gets no devbox shellrc (the case lists `"shell"` instead of `"sh"`) |
| 3 | Medium | run | `devbox run` changes arguments (`strconv.Quote` and an unquoted `eval`) |
| 4 | Medium | lock/nix | `NewStorePathParts` panics on short paths and misreads prerelease versions |
| 5 | Medium | config | `LoadConfigFromURL` ignores the HTTP status, so `global pull` can save a 404 body as the config |
| 6 | Medium | nix | `HashFromNixPkgsURL` keeps the `?narHash=` query string in the commit hash |
| 7 | Medium | install | A selected output missing from the lockfile is a hard error instead of a fallback |
| 8 | Medium | devpkg | `Package.Equals` returns true for any two unresolved devbox packages |
| 9 | Low-Med | shell | Paths are written into the shellrc unescaped (`$`, `'`) |
| 10 | Low-Med | shell | `exportify` writes bash-only syntax (`export -f`) for dash and fish |
| 11 | Low-Med | scripts | Script names are used as file paths unchecked (`/`, `..`) |
| 12 | Low | launcher | A concurrent download deletes another launcher's finished install |
| 13 | Low | shellgen | The HEAD response body in the nixpkgs mirror check is never closed |
| 14 | Low | generate | Generated `.envrc` is nondeterministic and its values are unquoted |
| 15 | Low | services | `devbox services ls` order is random (it sorts into a map) |
| 16 | Low | info | `devbox info` reports network failures as "package not found" |
| 17 | Low | shell | Temp dirs and file handles leak per `devbox shell` |
| 18 | Low | update | `devbox update nixpkgs nixpkgs` updates every package |
| 19 | Low | nix | The "nix not found in $PATH" message is always overwritten |
| 20 | Low | nix | `parseSearchResults` panics on malformed output |
| 21 | Low | config | Removing a package from a legacy array config can delete the wrong entry |
| 22 | Low | lock | A `devbox.lock` with `null` values panics instead of returning an error |
| 23 | Low | searcher | Search-service HTTP calls have no timeout |
| 24 | Low | config | `setStringField` adds `"key": ""` when it should remove the field |
| 25 | **High** | cli | `devbox run -lq …` panics in `ShorthandLookup` (multi-char shorthand) |
| 26 | Medium | cli | Recovered panics make the CLI exit with status 0 |
| 27 | Medium | plugin | `devbox update` never clears the git plugin cache |
| 28 | Medium | plugin | Removing `pkg@ver` leaves its virtenv and plugin files behind |
| 29 | Medium | services | Every process-compose exit 1 is reported as "terminated remotely" |
| 30 | Low | services | `StopProcessManager` drops the deferred write error |
| 31 | Low | services | `lockFile` leaks a goroutine and a lock on timeout |
| 32 | Low | cli | `integrate vscode --debugmode` calls `log.Fatal` when the log file is missing |
| 33 | Low | cli | `devbox setup nix` reinstalls even when nix is present, and `--daemon` is unused |
| 34 | Low | pullbox | `global pull` wipes the profile before copying, and leaks temp dirs |

### Details

#### 1. High: GitHub token leaks into errors, logs and telemetry
- **Where:** `internal/nix/command.go:16-19` and `nix/command.go:339-351`.
- **Cause:** `init()` appends `--option access-tokens github.com=$GITHUB_TOKEN` to every nix command's arguments. `Cmd.error()` builds the message as `"nix: command error: " + c.String()`, which contains the full argument list, and `cmdError.Redact()` returns the same text. `CurrentConfig` and `DaemonVersion` also mark the command `redact.Safe`.
- **Impact:** Any failing nix command prints the token to stderr, debug logs and telemetry. The token is also visible to other local users through `ps`.
- **Confirmed:** With `GITHUB_TOKEN=ghp_SECRETTOKEN`, a failing command's error contained `'github.com=ghp_SECRETTOKEN'`.
- **Fix:** Pass the token through the command's environment as `NIX_CONFIG=access-tokens = github.com=…`, or mask it in `Args.String()`, `LogValue` and `Redact`.

#### 2. High: `SHELL=/bin/sh` gets no devbox setup
- **Where:** `internal/devbox/shell.go:166`.
- **Cause:** The case reads `case "dash", "ash", "shell":`. `sh` is not listed, so `/bin/sh` falls into `shUnknown` and no `ENV` rc file is set.
- **Impact:** In Docker, Alpine and busybox images and in CI, `devbox shell` runs without init hooks, the prompt, aliases or `refresh`.
- **Fix:** Change `"shell"` to `"sh"` and add a test for it.

#### 3. Medium: `devbox run` changes arguments
- **Where:** `internal/devbox/devbox.go:351, 374, 384`.
- **Cause and impact:**
  - Arguments are quoted with `strconv.Quote`, which writes Go escapes, so a real newline in an argument becomes a literal `\n`.
  - The wrapper runs `eval $DEVBOX_RUN_CMD` unquoted, which word-splits it, so `devbox run echo 'a   b'` prints `a b`.
- **Fix:** Use `eval "$DEVBOX_RUN_CMD"`, and a POSIX double-quote escaper that escapes only `"`, `\` and backtick.

#### 4. Medium: `NewStorePathParts` panics or misparses
- **Where:** `internal/nix/storepath.go:27-38`.
- **Panic:** `path[:32]` has no length check. A lockfile output with an empty `path` (the field is `omitempty`) crashes the process, including from `FillNarInfoCache` goroutines.
- **Misparse:** `linux-6.10-rc7` parses to version `6.10`, output `rc7`. `FixMissingStorePaths` then writes that output name into `devbox.lock`.
- **Fix:** Validate the length and the separator. Only treat known output names as outputs.

#### 5. Medium: `LoadConfigFromURL` ignores the HTTP status
- **Where:** `internal/devconfig/config.go:197-213`.
- **Cause:** The body is parsed whatever the status code. A 404 with a JSON body such as `{"message":"Not Found"}` parses as an empty config.
- **Impact:** `devbox global pull <url>` silently replaces the user's global config with that empty config.
- **Fix:** Return an error for any non-2xx status.

#### 6. Medium: `HashFromNixPkgsURL` returns the commit plus a query string
- **Where:** `internal/nix/nixpkgs.go:127-131`.
- **Cause:** The regex `([^#]+)` captures `?lastModified=…&narHash=…` as part of the hash.
- **Impact:**
  - `EnsureNixpkgsPrefetched` caches under a key that never matches, so it prefetches again.
  - It prefetches `github:…/<rev>?narHash=…` without `FixInstallableArg`. That combination is known to break on Nix < 2.25.
- **Fix:** Capture `[^#?]+`, or use `installable.Ref.Rev`.

#### 7. Medium: A missing lockfile output is a hard error
- **Where:** `internal/devpkg/narinfo_cache.go:175` and `internal/lock/package.go:85`.
- **Cause:** When the lockfile has the system but not the selected output, `Output %s not found` is returned. A typical case is a legacy `store_path`-only entry combined with `outputs: ["dev"]`.
- **Impact:** Install fails instead of falling back to the "not in cache" path.
- **Fix:** Treat a missing output as "not cached" (return `nil, nil`).

#### 8. Medium: `Package.Equals` is wrong for unresolved packages
- **Where:** `internal/devpkg/package.go:482`.
- **Cause:** `installable` is empty until `resolve()` runs, so `Equals("go@1.20", "python@3.11")` returns `true`.
- **Impact:** Limited today, since the only caller is `nixprofile.Matches`.
- **Fix:** Resolve both packages first, or short-circuit only when both installables are set.

#### 9. Low-Medium: Unescaped paths in the generated shellrc
- **Where:**
  - `internal/devbox/shellrc.tmpl:21-29, 47, 65`
  - `shellrc_fish.tmpl:59`
  - `refresh.go:45`
- **Cause:** `ProjectDir`, `HooksFilePath`, `HistoryFile` and `OriginalInitPath` are written raw inside `"…"`.
- **Impact:**
  - A project dir containing `$` makes `cd` fail, and the shell exits.
  - A dir containing `'` breaks the `refresh` alias with a syntax error.
- **Fix:** Shell-quote these values before passing them to the templates, with a separate quoter for fish.

#### 10. Low-Medium: `exportify` writes bash syntax for every shell
- **Where:** `internal/devbox/envvars.go:71-80, 102`.
- **Cause:** Exported bash functions (`BASH_FUNC_x%%`, for example Lmod's `module`) are emitted as a function definition plus `export -f`.
- **Impact:**
  - dash fails with `export: Illegal option -f`.
  - fish cannot parse the output.
  - Backtick escaping follows POSIX rules, so values containing a backtick are wrong in fish.
- **Fix:** Emit functions only for bash, and add a fish-specific exporter.

#### 11. Low-Medium: Script names are used as file paths
- **Where:** `internal/shellgen/scripts.go:172`.
- **Cause:** `validateScripts` only rejects whitespace.
- **Impact:**
  - A script named `lint/fix` breaks every `shell` and `run`, because the subdirectory does not exist.
  - A name like `../x` writes outside the scripts directory.
  - The names `.hooks` and `.cmd` collide with reserved files.
- **Fix:** Reject `/`, `..` and the reserved names.

#### 12. Low: A concurrent download removes a finished install
- **Where:** `internal/launcher/install.go:155`.
- **Cause:** `download` runs `os.RemoveAll(dir)` before `Rename`, even when another launcher has just completed the same install.
- **Impact:** The other launcher's `exec` can fail with ENOENT.
- **Fix:** Return early if the binary is already present and executable.

#### 13. Low: Unclosed HTTP response
- **Where:** `internal/shellgen/nixpkgs.go:43`.
- **Cause:** The `client.Head(mirrorURL)` response body is never closed. This runs for every flake input whenever `DEVBOX_CACHE` is set.
- **Fix:** Add `defer resp.Body.Close()` once `err == nil`.

#### 14. Low: Nondeterministic and unquoted `.envrc`
- **Where:** `internal/devbox/generate/devcontainer_util.go:164, 273`.
- **Cause and impact:**
  - It ranges over `EnvMap` unsorted, so the `--env` order changes on every run and creates git noise.
  - Values and `--config <dir>` are unquoted, so a space breaks the `eval`.
- **Fix:** Sort the keys and shell-quote the values.

#### 15. Low: Random service order
- **Where:** `internal/devbox/devbox.go:673` and `internal/devbox/services.go:144`.
- **Cause:** `Services()` sorts the keys, then inserts them into a map, so the sort has no effect. `ListServices` ranges over that map.
- **Fix:** Sort the names where `ListServices` iterates over them.

#### 16. Low: `devbox info` reports network failures as "not found"
- **Where:** `internal/devbox/devbox.go:486`.
- **Cause:** The condition `!errors.Is(err, searcher.ErrNotFound)` returns "Package %q not found" for every error *except* not-found, including when the search service is down.
- **Fix:** Return a lookup-failure error in that branch.

#### 17. Low: Temp dir and file handle leaks
- **Where:** `internal/devbox/shell.go:286, 451`.
- **Cause and impact:**
  - Each `devbox shell` creates an `os.MkdirTemp` directory and never removes it. It holds a copy of the user's rc file and the exported environment.
  - `defer file.Close()` inside a loop keeps every file open until the function returns.
- **Fix:** Remove the directory once the shell exits, and close each file inside the loop.

#### 18. Low: Duplicate `nixpkgs` arguments update everything
- **Where:** `internal/devbox/update.go:29-38`.
- **Cause:** `devbox update nixpkgs nixpkgs` skips the `len == 1` early return. `DeleteFunc` then leaves an empty list, which means "update everything".
- **Fix:** Return when the list is empty after `DeleteFunc`.

#### 19. Low: The "nix not found in $PATH" message is overwritten
- **Where:** `nix/command.go:339-349`.
- **Cause:** The `switch` that follows always reassigns `cmdErr.msg`, so the not-found message never shows.
- **Fix:** Run the switch only when `msg` is still empty.

#### 20. Low: `parseSearchResults` panics
- **Where:** `internal/nix/search.go:48, 54-55`.
- **Cause:** It calls `panic(err)` on invalid JSON and uses unchecked `.(string)` type assertions.
- **Fix:** Return errors and use checked type assertions.

#### 21. Low: Legacy array config removes the wrong package
- **Where:** `internal/devconfig/configfile/packages.go:38-46` and `ast.go:167-173`.
- **Cause:** The syntax tree removes the first element whose name matches.
- **Impact:** With `["python@3.10","python@3.11"]`, removing `python@3.11` deletes `python@3.10` from `devbox.json`.
- **Fix:** Match on version as well as name.

#### 22. Low: A null lockfile crashes
- **Where:** `internal/lock/lockfile.go:109` and `internal/lock/package.go:140`.
- **Cause:**
  - `"packages": null` leaves a nil map, so `Resolve` panics on assignment.
  - A `null` system entry or package entry dereferences nil.
- **Fix:** Normalize the data in `GetFile`.

#### 23. Low: No timeout on search-service calls
- **Where:** `internal/searcher/client.go:180` and `internal/lock/resolve.go:68`.
- **Cause:** Both use `context.TODO()` with `http.DefaultClient`.
- **Impact:** A stalled connection hangs `add` and `install` forever, and the retry logic never runs.
- **Fix:** Add a per-request deadline.

#### 24. Low: `setStringField` writes an empty field
- **Where:** `internal/devconfig/configfile/field.go:44-48`.
- **Cause:** When the key is missing and the value is empty, it appends `"key": ""`. The branch for an existing key removes the field instead.
- **Fix:** Do nothing when the key is absent and the value is empty.

#### 25. High: `devbox run` panics on combined short flags
- **Where:** `internal/boxcli/run.go:301`.
- **Cause:** `wrapArgsForRun` calls `runFlags.ShorthandLookup(strings.TrimLeft(arg, "-"))`. pflag panics when the name is longer than one character.
- **Impact:**
  - `devbox run -lq foo` and `devbox run -cdir script` crash before cobra parses anything.
  - Because of #26, the crash exits 0.
- **Confirmed:** Running `devbox run -lq foo bar` printed `can not look up shorthand which is more than one ASCII character` and exited with status 0.
- **Fix:** Look up only `arg[1:2]` for single-dash arguments, or return `args` unchanged when the name is longer than one character.

#### 26. Medium: Panics exit with status 0
- **Where:** `internal/boxcli/root.go:103-104`.
- **Cause:** `Execute` has an unnamed `int` result and uses `defer debug.Recover()`, so after a recovered panic it returns 0.
- **Impact:** CI and scripts treat crashes as successes.
- **Fix:** Use a named return value and set it to 1 in the recover handler.

#### 27. Medium: `devbox update` does not refresh `git:` plugins
- **Where:** `internal/plugin/update.go:4`, `git.go:18` and `cache.go:59-64`.
- **Cause:** `Update()` only clears `githubCache`, so `gitCache` stays stale for up to 24h.
- **Related:** `getOrFetch` treats any cache read error as fatal. `filecache.Set` writes non-atomically, so a truncated entry breaks plugin loading until the cache is cleared.
- **Fix:** Clear both caches, and treat an undecodable entry as a cache miss.

#### 28. Medium: Removing a versioned package leaves its plugin files
- **Where:** `internal/plugin/rm.go:16`, called from `internal/devbox/packages.go:373`.
- **Cause:** Removal uses `found.Raw` (`nodejs@20`), but the files were created under `CanonicalName()` (`nodejs`).
- **Impact:** `.devbox/virtenv/nodejs` and the `bin/` symlinks stay on PATH. The same happens for python and ruby.
- **Fix:** Pass `CanonicalName()`, and check that the resulting path stays under `VirtenvPath`.

#### 29. Medium: Real process-compose failures are hidden
- **Where:** `internal/services/manager.go:201`.
- **Cause:** `waitErr.Error() == "exit status 1"` is treated as "terminated remotely" and returns nil. process-compose also exits with 1 for a bad YAML file or a port it can't bind.
- **Impact:** `services up` reports a remote stop and exits 0. `cleanupErr` is also dropped.
- **Fix:** Detect a remote stop explicitly, and return `waitErr` otherwise.

#### 30. Low: `StopProcessManager` loses the write error
- **Where:** `internal/services/manager.go:281-284`.
- **Cause:** The deferred `err = writeGlobalProcessComposeJSON(...)` assigns to a local variable, because the return value is unnamed.
- **Impact:** If the write fails, the stale PID entry stays, and the next `services up` reports "already running".
- **Related:** `StopAllProcessManagers` (line 310) prints with `fmt.Printf`, without a newline, to stdout rather than `w`.
- **Fix:** Use a named return and set it from the deferred write; print to `w` with a newline.

#### 31. Low: `lockFile` leaks a goroutine and a lock on timeout
- **Where:** `internal/services/manager.go:391-412`.
- **Cause:** `flock` runs in a goroutine that sends on an unbuffered channel. After the 5s timeout the caller closes the file and returns, so:
  - the goroutine blocks forever, or takes a lock on a file descriptor that has been closed and possibly reused.
- **Fix:** Use a buffered channel, or `LOCK_NB` with a retry loop until the deadline.

#### 32. Low: `--debugmode` exits when the log file is missing
- **Where:** `internal/boxcli/integrate.go:183`.
- **Cause:** The log file is opened without `O_CREATE`, and an error calls `log.Fatal`. The path is relative to the current directory, not the project.
- **Fix:** Add `O_CREATE`, build the path from the project dir, and log errors instead of exiting.

#### 33. Low: `devbox setup nix` reinstalls nix
- **Where:** `internal/boxcli/setup.go:43-51` and `internal/nix/install.go:67`.
- **Cause:** There is no `return` after the "already installed" message, so the installer runs anyway. `--daemon` is never read.
- **Fix:** Return after the message, and either use the daemon flag or remove it.

#### 34. Low: `global pull` can lose data
- **Where:** `internal/pullbox/files.go:44`, `pullbox.go:60, 82` and `config.go:37`.
- **Cause:** `ClearDir(ProjectDir())` runs before the copy.
- **Impact:**
  - If the copy fails, the profile has already been wiped.
  - If the source is inside the profile, the source is deleted before it is copied.
  - Temp dirs from clone and extract are never removed.
- **Fix:** Copy into a staging directory and swap it in, and `defer os.RemoveAll(tmp)`.

### Minor items (not counted)
- `internal/fileutil/untar.go:66` calls `log.Fatal` from inside a library function.
- "Skipping" errors in `plugin/services.go` and `services/config.go` have no trailing newline.
- `services/client.go` sends HTTP requests with no timeout, and does not URL-escape service names.

### Checked and found clean
- **Untar:** the zip-slip and symlink checks are correct.
- **vercheck:** the version comparisons are correct.
- **patchpkg:** the slice bounds are safe.
- **Templates:** the built-in plugin templates only use keys that are provided.
- **Locking and cache keys:** the lockfile locking, singleflight use, narinfo cache concurrency and cache-key hashing are correct.
- **Other:** searcher retries, the state-hash comparison and `internal/envir` have no issues.

## Suggested fix order
1. **Quick, high-value fixes:**
   - #1 token leak
   - #25 and #26 (crash, then exit 0)
   - #2 `sh` shellrc
2. **Data-loss and correctness:**
   - #5 and #34 (`global pull`)
   - #4 and #22 (lockfile panics)
   - #3 `run` quoting
   - #27 and #28 (plugin cache and leftover files)
   - #29 hidden service failures
3. **The rest:** each is a few lines and can be batched.
