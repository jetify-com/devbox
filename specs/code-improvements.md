# Devbox code simplification opportunities

A list of places where Devbox code can be made simpler or easier to read, or
deleted outright. Findings come from reading `internal/`, `cmd/` and `pkg/` and
grepping every caller across the repo (Go code, `.tmpl` files, testscripts and
the VS Code extension). The module is on **Go 1.27**, so `slices`, `maps`,
`cmp`, `strings.Cut`/`CutPrefix`, `sync.OnceValue` and iterators are all
available.

Sizes are approximate **lines removed or rewritten**. Risk is **low** unless
noted.

## Summary

| Tier | Theme | Approx. lines |
|---|---|---|
| A | Dead code: no callers, or only tests call it | ~750 |
| B | Code unreachable or flag-gated behind settings that are on by default | ~250 |
| C | Duplicated logic that should share one helper | ~550 |
| D | Small stdlib and readability cleanups | ~200 |
| E | Latent bugs found along the way | small |

Do tier A first: it is mechanical, low-risk, and makes tiers B and C easier
to review.

---

## A. Dead code (delete)

Every item was checked with a repo-wide grep. "Test-only" means the only
callers are the item's own tests, so delete the tests too.

### nix / nixprofile

- **A1. `nixprofile` lookup API** (`internal/nix/nixprofile/profile.go:210-260`, `item.go:42-86`).
  - Dead: `ProfileListNameOrIndexArgs`, `ProfileListNameOrIndex`, and `NixProfileListItem.Matches`, `MatchesUnlockedReference` and `addedByStorePath`.
  - Test-only: `AttributePath`.
  - Once these go, `devpkg.Package.Equals` (`internal/devpkg/package.go:478-499`) loses its only caller and can go too.
  - The only API used outside the package is `ProfileListItems`. Keep `NameOrIndex`, which `String()` uses.
  - ~110 lines.
- **A2. `nix.DaemonVersion` / `DaemonError`** (`internal/nix/store.go:117-199`): no callers. ~80 lines.
- **A3. `internal/nix/config.go`** (`CurrentConfig`, `Config`, `ConfigField`, `IsUserTrusted`): test-only. Delete it with `config_test.go`. ~105 lines plus the test.
- **A4. `internal/nix/writer.go`** (`PackageInstallWriter`): no references. ~46 lines.
- **A5. Misc nix helpers.**
  - `eval.go`: `PackageIsInsecure`, `IsInsecureAllowed`, and `Eval`, which only those two call.
  - `nix.go:129`: `SystemIsLinux`.
  - `search.go:19`: `ErrPackageNotInstalled`.
  - ~60 lines.

### devpkg / devconfig / plugin

- **A6. devpkg nixpkgs prefetch path** (`internal/devpkg/package.go:532-580`).
  - Dead: `EnsureNixpkgsPrefetched`, `(*Package).ensureNixpkgsPrefetched` and `(*Package).HashFromNixPkgsURL`.
  - `internal/devpkg/validation.go:32-48`: `ValidateInstallsOnSystem`.
  - ~50 lines.
- **A7. `internal/devconfig/configfile/field.go`** (`SetStringField`, which uses reflection) plus the unused `type Stage` (`file.go:93-96`). ~60 lines.
- **A8. `LocalPlugin.IsLocal`** (`internal/plugin/local.go:41`): unused.
- **A9. `LocalPlugin.addFilenameIfMissing`** (`local.go:68-73`): a no-op, because `Path()` already appends `plugin.json`.

### devbox / shellgen / lock

- **A10. The "set by devbox" env marker** (`internal/devbox/envvars.go:19,189-209`, called at `devbox.go:823-825`).
  - `markEnvsAsSetByDevbox(configEnv)` writes `__DEVBOX_SET_*` keys into a local map that is never read again.
  - So `addEnvIfNotPreviouslySetByDevbox` is effectively `maps.Copy(env, configEnv)`.
  - Replace it with `maps.Copy`, and delete both helpers and the constant. ~25 lines.
- **A11. `Devbox.environment` / `validateEnvironment`** (`devbox.go:62,139-147,1350-1361`).
  - The field is set and never read. It is left over from Jetify Cloud secrets.
  - Drop it. Then deprecate or hide the `--environment` flag (`boxcli/config.go:23,30`), whose help text still promises "secrets support".
  - ~20 lines in devbox. Removing the flag is user-visible, so deprecate it rather than delete it.
- **A12. shellgen template plumbing.**
  - `generate.go:155-167`: `toJSON` and `templateFuncs` (`json`, `contains`, `debug`) are not used by any template in `tmpl/`.
  - `flake_input.go:25-27`: `flakeInput.HashFromNixPkgsURL` has no callers.
  - `nixpkgs.go:15-31`: `NixpkgsInfo.TarURL` is unused. Collapse the struct to `func nixpkgsURL(rev string) string`.
  - ~30 lines.
- **A13. lock / envpath.**
  - Unused: `lock.File.Remove` (`lockfile.go:78-85`) and `envpath.RemoveFromPath` (`pathlists.go:40-57`).
  - Test-only: `IsUpToDateAndInstalled` (`lockfile.go:277-283`). Point the test at `StateStatus`.
  - Single caller: `File.Add` (`lockfile.go:69-76`). Inline it into `UpdateStdenv`.
  - ~40 lines.

### boxcli / services / ux / envir

- **A14. `usererr`** (`internal/boxcli/usererr/usererr.go`).
  - `NewLogged`, `NewWarning` and `WithLoggedUserMessage` have no callers. As a result, `level` is always error and `logged` is always false.
  - Delete them, plus `IsWarning`, the `level` type and the `logged` field.
  - Delete the warning branch in `midcobra/debug.go:62-65`.
  - `ux.Fwarning` then has no callers either.
  - ~50 lines.
- **A15. Dead `ux` / `envir` / `cmdutil` symbols.**
  - `ux.Fsuccess` and `ux.Ferror`.
  - `envir.DevboxGateway`, `envir.DevboxVM` and `envir.User`.
  - `envir.GetValueOrDefault`, which has one caller: use `cmp.Or(os.Getenv(k), def)`.
  - `cmdutil.Exists`, which has one caller: inline it.
  - ~25 lines.
- **A16. Feature-flag helpers** `EnableOnDev` and `EnableForTest` (`boxcli/featureflag/feature.go:68,75`): no callers.
- **A17. `services` dead state.**
  - `globalProcessComposeConfig.Path` (`internal/services/manager.go:39`) is written and never read.
  - Several `ctx` and `w` parameters are unused: `client.go` `Start/Stop/RestartServices` and `ListServices`; `manager.go` `StopProcessManager`, `StopAllProcessManagers` and `AttachToProcessManager`.
- **A18. `shellFlagDefaults`** (`boxcli/shell.go:28-33`): only ever passed as the zero value. Delete the type and the parameter.
- **A19. `versionInfo` / `getVersionInfo`** (`boxcli/version.go:79-100`): one caller, and `IsPrerelease` is never set. Inline it. ~20 lines.
- **A20. `integrate` command** (`boxcli/integrate.go`).
  - The `--config` flag is never read; the directory comes over IPC.
  - The parent command's `Args`, `PreRunE` and `RunE: cmd.Help` do nothing. `PreRunE` isn't persistent, and cobra shows help by default.
- **A21. `generate` parent** (`boxcli/generate.go:49,71`): it registers a `--config` flag that nothing reads, because the parent has no `RunE`.

### Other packages

- **A22. Delete the `internal/goutil` package.** `PickByKeysSorted` has no callers.
- **A23. `fileutil`.**
  - Unused: `IsDir`, `EnsureDirExists` and `EnsureAbsolutePaths`.
  - Test-only: `IsDirEmpty`.
  - ~55 lines plus ~70 of test.
- **A24. `pullbox/tar.Compress`** (`internal/pullbox/tar/tar.go:54`): no callers.
- **A25. `patchpkg.NewDerivationBuilder`** (`internal/patchpkg/builder.go:57`): no callers. It is also wrong in principle, because `Build` calls `init()` again.
- **A26. `debug.SetOutput`** (`internal/debug/debug.go:40`): no callers.
- **A27. `cuecfg`.**
  - `InitFile` has no callers.
  - TOML and XML support (`toml.go`, `xml.go`) is never exercised: no TOML or XML file is read or written. A `.toml` or `.xml` pull URL is accepted, then fails, because the loader only parses JSON.
  - Delete both formats. Unexport or inline `ParseFileWithExtension`, `Marshal` and `Unmarshal`.
  - Consider renaming the package, since it contains no CUE code.
  - ~65 lines. Risk: low to medium, because those URLs would now be rejected up front instead of failing later.

---

## B. Unreachable or always-on paths

- **B1. `--daemon` / `nixDaemonFlagVal` is a no-op.**
  - `nix.EnsureNixInstalled(ctx, w, withDaemonFunc)` (`internal/nix/install.go:67`) never calls `withDaemonFunc`, and `runInstallNixCmd` ignores `--daemon` as well.
  - Drop the parameter, `nixDaemonFlagVal` (`boxcli/setup.go:57-85`, including its "running as root" warning) and the constant.
  - Then either delete the `--daemon` flag or wire it into the installer. **Ask the owners which they intended.**
  - ~35 lines.
- **B2. Code for Nix older than the minimum.** `MinVersion` is 2.18 (`internal/nix/shim.go:27`), and `EnsureNixInstalled` rejects anything older, so these branches never run:
  - The non-JSON `nix profile list` parser: `profileListLegacy` and `parseNixProfileListItemLegacy` (`profile.go:175-208,262-298`), and the `useJSON` parameter of `nix.ProfileList`. Keep the legacy *JSON-array* format, because 2.18 and 2.19 still print it.
  - `AtLeast(2_14)` in `store.go:152` (moot once A2 is done) and `!AtLeast(Version2_17)` in `narinfo_cache.go:209`.
  - Unused version constants `Version2_12`, `2_13`, `2_15`, `2_16`, `2_18` and `2_21`–`2_24`.
  - ~90 lines. Risk: medium, but only if some code path skips `EnsureNixInstalled`.
- **B3. The `ResolveV2` feature flag is on by default.**
  - The v1 path in `internal/lock/resolve.go:43-79` and `buildLockSystemInfos` (`:147-191`) only runs with `DEVBOX_FEATURE_RESOLVE_V2=0`.
  - Remove the flag and the v1 path. `boxcli/search.go:52` still uses v1 `Resolve` for `pkg@ver` search, so move it to v2 or keep that client method.
  - If v1 stays, at least tidy it up:
    - remove the obsolete loop-variable copies;
    - use a plain `Mutex` instead of the `RWMutex`;
    - drop the duplicated `len(outputs)` check in `resolveV2`.
  - ~60 lines plus tests (`lock/lockfile_test.go`, `testscripts/testrunner/run_test.test.txt`). Risk: medium, because it removes a user escape hatch.
- **B4. `ScriptExitOnError`** is also on by default. **Keep it.** It is the only way for users to opt out, so it is listed here only for completeness.
- **B5. Hidden dev-only commands.** `generate debug` (`generate.go:76`) and the root `all` command (`root.go:85-93,142-150`) have no tests, docs or scripts. Consider deleting them. ~30 lines. Risk: medium, because maintainers may use them by hand.
- **B6. Unreachable fallback in `DevboxShell.shellPath`** (`internal/devbox/shell.go:131-137`). The `bashNixStorePath != ""` / `ErrNoRecognizableShellFound` branch can't be reached; `nix build` would already have failed.

---

## C. Duplication: consolidate into one helper

### boxcli

- **C1. The repeated `devbox.Open` block.**
  - `devbox.Open(&devopt.Opts{Dir: flags.config.path, Environment: flags.config.environment, Stderr: cmd.ErrOrStderr()})` plus the `errors.WithStack` wrapping appears about 20 times: add, info, rm, install, pull, push, update, generate ×4, run ×2, shell, shellenv and services ×7.
  - Add `func (f *configFlags) open(cmd *cobra.Command, env map[string]string) (*devbox.Devbox, error)`.
  - This also fixes `list.go:31` and `generate alias` (`generate.go:229`), which silently drop `--environment`.
  - **~100 lines. This is the single biggest win.**
- **C2. Seven services handlers** (`boxcli/services.go:154-296`). attach, ls, start, stop, restart, up and pcport each re-parse env and reopen the box. With C1 plus a `withEnv` switch, each handler shrinks to about 3 lines. ~80 lines.
- **C3. Duplicated env options in shell, run and shellenv.** `shell.go:101-117`, `run.go:194-210` and `shellenv.go:139-156` build the same `devopt.EnvOptions`, differing only in the hint text. Add one `envOptions(...)` helper. ~25 lines.
- **C4. `installCmdFlags` embeds all of `runCmdFlags`** (`install.go:18`) but only uses `.config`. Change it to `config configFlags`. That also fixes `pull` dropping `--environment`.
- **C5. `generate.go`.**
  - `runGenerateCmd` (`:283-302`) dispatches on `cmd.Use` strings. Call `box.Generate` and `box.GenerateDevcontainer` directly from each `RunE`.
  - Give each subcommand its own minimal, unexported flag struct.
  - Hoist the `regexp.MustCompile` (`:239`) to a package-level variable.
- **C6. `configFlags` / `pathFlag`** (`boxcli/config.go`). `register` and `registerPersistent` are duplicated. Take a `*pflag.FlagSet` instead; that also fixes the help-text drift "e.g.secrets" vs "e.g. secrets". Flatten `pathFlag` into `configFlags`.

### services

- **C7. `services/client.go:31-80`.** `Start/Stop/RestartServices` differ only by verb and HTTP method. Collapse them into one helper. Use `http.NewRequestWithContext` (so `ctx` is finally used) and `io.ReadAll`. ~45 lines.
- **C8. `services/manager.go`.**
  - The "open, lock, read, defer close" preamble appears 6 times. Extract `withGlobalConfig(func(cfg, file) error) error`.
  - `exec.Command` is built twice, and the `instance{Pid, Port}` write is duplicated between the foreground and background start (`:163-226`).
  - `append(requestedServices, flags...)` (`:139-150`) can alias the caller's slice.
  - ~60 lines. Risk: low to medium, because it touches lock ordering.

### devbox / shellgen

- **C9. runx install logic exists twice.** `devbox.go:1263-1303` (`RunXPaths`) and `packages.go:644-658` (`InstallRunXPackages`) both resolve and install every runx package. Each package is installed twice per `ensureStateIsUpToDate`. Use one `runXInstall(ctx) ([]string, error)` and unexport the function. ~15 lines.
- **C10. `mergeResolvedPackageToLockfile`** (`internal/devbox/update.go:282-305`). `SystemInfo.Equals` is nil-safe, so both branches reduce to `!info.Equals(existing.Systems[sys])` in a single loop. ~15 lines; `update_test.go` covers it.
- **C11. Devcontainer and envrc generation** (`internal/devbox/generate/devcontainer_util.go`).
  - `CreateEnvrc` (`:161`) and `EnvrcContent` (`:271`) build `--env k=v` flags separately, by ranging over a map, so the order is non-deterministic. Use one helper with sorted keys.
  - Hoist the regexp (`:242`) to a package-level variable.
  - `CreateDevcontainer` becomes `json.MarshalIndent` plus `os.WriteFile`.
  - `Devbox.PrintEnvrcContent` is just a wrapper; boxcli can call `generate.EnvrcContent` directly.
- **C12. `BuildInputs` / `BuildInputsForSymlinkJoin`** (`internal/shellgen/flake_input.go:48-137`).
  - Both repeat the nixpkgs-vs-flake attribute-path mapping. Extract `inputAttrPath`, and replace the `lo.Map`-with-captured-`err` with a plain loop.
  - Drop the resolvable uniqueness TODO at `:189-193`.
- **C13. "Write flake.nix and delete flake.lock if it changed"** is implemented twice: `shellgen/generate.go:169` `makeFlakeFile` and `flake_plan.go:319-337` `writeTo`. Share one.

### devpkg / devconfig / plugin

- **C14. `configfile.PackagesMutator`** (`internal/devconfig/configfile/packages.go`).
  - The "parse name, find index, else error 'not found'" block appears 6 times. Add `find(name) (*Package, error)`.
  - `SetOutputs` and `SetAllowInsecure` are the same, and so are `AddPlatforms` and `ExcludePlatforms` (it even has a `//nolint:dupl`).
  - `ast.go:196-218`: `appendPlatforms`, `appendOutputs` and `appendAllowInsecure` are identical wrappers around `appendStringSliceField`.
  - ~55 lines.
- **C15. devpkg binary-cache helpers** (`package.go:584-635`, `narinfo_cache.go:28-49`).
  - `IsInBinaryCache` and `IsOutputInBinaryCache` have identical bodies.
  - `InputAddressedPaths` re-checks the cache and re-resolves the lockfile for every output.
  - `InputAddressedPathForOutput` hand-loops where `SystemInfo.Output(name)` already exists.
  - ~30 lines.
- **C16. devpkg option application.**
  - `PackagesFromConfig` and `PackageFromStringWithOptions` (`package.go:107-131`) set the same four fields. Extract `applyOpts`.
  - `LegacyToVersioned` is equivalent to `Versioned()` at its only call site (`devbox/update.go:111`).
- **C17. `devconfig.Config`** (`internal/devconfig/config.go`).
  - `Hash` and `NixHash` (`:410-445`) differ only in the root-hash call.
  - The built-ins loop in `loadRecursive` (`:276-292`) duplicates the include recursion.
  - ~25 lines.
- **C18. `plugin`.**
  - `createFile` and `buildConfig` (`plugin.go:221-261`) build the same five template variables.
  - `getConfigIfAny` (`files.go:21-32`) has identical `*githubPlugin` and `*gitPlugin` cases.
  - `gitPlugin` strips `?query` in two places (`git.go:47,79`); use one `strings.Cut`.
  - Make `isFullCommitHash` a positive hex check instead of `!isBranchName`.
  - `GetServices` always returns a nil error.
  - ~40 lines.

### Cross-package helpers

- **C19. Atomic file writes.** `launcher.writeVersion` (`internal/launcher/files.go:126`), `launcher.replaceFile` (`replace.go:56`) and `shellgen.writeScriptFileAtomic` (`scripts.go:116,158`) each do temp file, write, chmod and rename. Add `fileutil.WriteFileAtomic`. The shellgen copy must keep holding `ForkLock`. ~30 lines.
- **C20. "Copy each entry with `cp -r`" loops.** These appear in `fileutil.CopyAll` (`dir.go:16`), `templates.copyDirContents` (`template.go:63`, which skips `.git`) and `pullbox.copyToProfile` (`files.go:21-57`). Give `CopyAll` a `skip ...string` parameter and use it in all three. ~40 lines.
- **C21. `withEnv` is duplicated** in `launcher/launcher.go:227` and `vercheck/project.go:203`. Keep one, for example `envir.WithEnv` next to `MapToPairs`. ~15 lines.
- **C22. Stack-trace walk.** `debug.EarliestStackTrace` and `telemetry.newSentryException` (`sentry.go:66-103`) declare the same tracer interfaces and walk the same unwrap chain. Have telemetry call `debug`.
- **C23. `telemetry.Event`** (`telemetry.go:169-190`). Three `switch` cases differ only by name; use a name map. `log.go:31-45` in boxcli has the same pattern.
- **C24. `pkg/autodetect`** (`autodetect.go:31-71`). `populateConfig` runs detection twice, once through `packages()` and once through `env()`. Detect once. `Detector.Relevance(path)` never uses `path`, so drop it. Note that `pkg/` is nominally public.
- **C25. Fold `internal/conf`** (one function, one caller at `devbox.go:1146`, with a stale package doc) into `devbox`.

---

## D. Stdlib and readability cleanups

- **D1. Sorted map keys by hand.**
  - `devbox.go:399` (`ListScripts`), `envvars.go:60,128` (`exportify*`), `shell.go:372` (`aliasLines`), `plugin.Config.Hash` and `templates.List` collect and sort keys manually. Each becomes `slices.Sorted(maps.Keys(m))`.
  - `devbox.go:666-678` (`Services`) sorts keys and then copies into another map, so the sort does nothing.
  - `templates.List` also sorts the package-level `popularTemplates` **in place**.
- **D2. Copy loops.** `devbox.go:796,898,1134`, `shell.go:236` and `boxcli/env.go:46` can use `maps.Copy`.
- **D3. Containment and dedupe loops.**
  - `HasDeprecatedPackages`, `isBeingUpdated` and `vercheck`'s `lo.ContainsBy` can use `slices.ContainsFunc`.
  - `IsEnabledOnPlatform` and `nix.EnsureValidPlatform` can use `slices.Contains`.
  - `run.go:134` `sort.Strings` plus `lo.Uniq` becomes `slices.Sort` plus `slices.Compact`.
- **D4. `golang.org/x/exp` imports.** `envpath/stack.go` imports `x/exp/slices` and `nix/store.go` imports `x/exp/maps`. Switch both to the stdlib.
- **D5. Go 1.22 loop-variable copies.** `narinfo_cache.go:63,71` and `lock/resolve.go:158` still copy loop variables.
- **D6. `strings.Cut` / `CutPrefix`.** Use them in `devbox/flakes.go:17-22` and the `plugin/git.go` query stripping.
- **D7. `sync.OnceValue`.** `build.OS` (`internal/build/build.go:50-71`) can use it. Unexport the `OS*` constants, which are only used inside the package.
- **D8. `setupShellStartupFiles`** (`devbox/shell.go:406-468`).
  - It re-parses a constant template on every iteration, and `defer`s `Close` inside the loop.
  - Use one `os.WriteFile(..., fmt.Appendf(...))`, and return early if the shell is not zsh. ~20 lines.
- **D9. Shell functional options with one caller** (`devbox/shell.go:71,181-204`). `newShell` is always called with all four options from `Devbox.Shell`, and `projectDir` is always `devbox.projectDir`. Set the fields directly. ~35 lines.
- **D10. Small inlines.**
  - `run.go` `parseScriptArgs` cannot fail; inline it as `args[0], args[1:]`. Also merge the two `allProjects` blocks.
  - `vercheck.printSuccessMessage`'s `toolName` is always `"Devbox"`.
  - `telemetry.initSentryClient`'s `appName` parameter and its panic can go.
  - `global.go:22` `filepath.Join(path)` is a no-op.
  - `RunScript` (`devbox.go:338`) uses a literal string where it should use `envir.DevboxShellEnabled`.
  - `pull.go:91` is a one-case `switch`; make it an `if`.
  - `gen-docs.go:36` walks parents by hand; use `cmd.Root()`.
  - `midcobra.getSubcommand`'s `TraverseChildren` branch never runs.
- **D11. devpkg tidy-ups.**
  - `ResolvedVersion` calls `lockfile.Get` twice.
  - `StoreName` can return `nix.EvalPackageName(u)` directly.
  - `pkgNeedsPatch` has two `slog.Debug` branches.
- **D12. `findPackageByName`** (`devbox.go:1040-1063`) uses a map of pointers plus `lo.Keys`. Use `lo.Filter`, which also gives a deterministic error order.
- **D13. `Info`** (`devbox.go:484-497`). The `packageVersion = nil` "fallthrough" branch is dead; collapse it to one `if err != nil`.
- **D14. `nix.ExperimentalFlags`.** Its only caller (`devbox/shell.go:111-125`) builds a raw `exec.Command("nix", ...)`. Use `nix.Command` there, which also picks up retries and `GITHUB_TOKEN`, then delete `ExperimentalFlags`. Risk: low to medium.
- **D15. Unexport single-package symbols.** These are only used inside their own package:
  - `SemverCompare` (vercheck)
  - `ParseRepoURL` (templates)
  - `LegacyPathInfo` (nix)
  - `InstallRunXPackages` (devbox)
  - the `GenerateReadmeCmdFlags` and `GenerateAliasCmdFlags` types (boxcli)
- **D16. `midcobra.Executable`** has one implementation and one caller. Return the concrete type.
- **D17. `pullbox` sub-packages.** `pullbox/git` and `pullbox/tar` each hold about one function; merge them into `pullbox`. `isModifiedConfig` repeats the `ValidNames` check that `profileIsNotEmpty` already did.

---

## E. Latent bugs found along the way

These are not simplifications, but most are fixed as a side effect of the
items above.

1. **`services.StopProcessManager` swallows a write error** (`internal/services/manager.go:281-284`). The deferred closure assigns to `err`, but the return value isn't named, so the assignment is lost. Fixed by C8.
2. **`midcobra/trace.go:20` asserts the wrong type**: `var _ Middleware = (*DebugMiddleware)(nil)` should be `(*TraceMiddleware)`.
3. **`DebugMiddleware.postRun` logs `exec.ExitError` twice** (`midcobra/debug.go:73-76`). Use if/else.
4. **`devbox.Open` can leave `box.stderr` nil** (`devbox.go:129-151`). It computes a defaulted `stderr` and then stores `opts.Stderr` instead.
5. **The generated `.envrc` / Dockerfile flag order is non-deterministic**, because it ranges over a map (C11).
6. **`setupShellStartupFiles` defers `Close` inside a loop** (D8).
7. **`pullbox` never deletes its temp dirs.** `git.CloneToTmp`, `git.Push`, `pullTextDevboxConfig` and `tar.Extract` each leave a directory behind on every pull or push.
8. **`--environment` is silently ignored** by `list`, `generate alias` and `pull`, because they don't pass it through (C1, C4). It is also a no-op everywhere else (A11).
9. **`--daemon` on `devbox setup nix` is ignored** (B1).
10. **`templates.List` mutates the global `popularTemplates`** by sorting it in place (D1).
11. **runx packages are installed twice per state recompute** (C9).

---

## Suggested PR breakdown

1. **Dead code sweep:** tier A, excluding A11 and A27. Mechanical, ~650 lines removed, low review cost.
2. **boxcli open helper and services handlers:** C1–C6 and A18–A21, with E8 fixed as a side effect. ~250 lines.
3. **services client and manager cleanup:** C7, C8 and A17, with E1 fixed. ~100 lines.
4. **Pre-2.18 Nix code removal:** B2. Needs a confirmation that every command path enforces `MinVersion`.
5. **devpkg / configfile / plugin consolidation:** C14–C18, D11.
6. **Cross-package helpers:** C19–C25.
7. **stdlib pass:** D1–D7. Easy to review with `go vet` and the tests.
8. **Decisions needed from maintainers first:** B1 (`--daemon`), B3 (`ResolveV2` v1 path), B5 (hidden commands), A11 (`--environment`) and A27 (TOML/XML pull).
