---
name: release-devbox
description: Cut a Devbox CLI release — recommend the version bump, write the release notes, and drive the draft or publish flow. Use when asked to release devbox, cut a version, ship 0.x.y, publish a release, or when a release is stuck partway through and needs diagnosing or resuming.
---

# Release the Devbox CLI

`scripts/release.ts` owns every mechanical step and the order they run in. Your
job is the judgment: read what's shipping, recommend a version, write the title
and notes, and help the user pick how to release.

Two entry points, both of which walk the whole pipeline:

| Command | What it does |
| --- | --- |
| `devbox run draft-release` | Builds the release, leaves it as a draft for review |
| `devbox run publish-release` | Builds and takes it live, **or** finishes an existing draft |

Read-only helpers: `devbox run release-changes` and `devbox run release-status`.

## 1. Look at what would ship

```bash
devbox run release-changes
```

This prints the commits since the last release grouped by kind (breaking /
features / fixes / other), the recommended version, and whether `flake.nix`
needs bumping. Read it before saying anything about the release.

The recommendation follows devbox's own history: it's pre-1.0, so **breaking
changes take a minor bump** (0.17.5 → 0.18.0) and **everything else takes a
patch bump** — 0.17.4 shipped new features as a patch. Say what the script
recommends and why, but sanity-check it against the actual diff. A release that
removes whole subsystems deserves a minor bump even if nobody wrote `feat!:`.

## 2. Ask how they want to release

Present the three options and let the user choose — don't assume:

1. **Draft first** (`devbox run draft-release`) — builds everything and stops at
   a draft. Right when the notes need review, or when the release is being
   coordinated with an announcement. This is the safe default.
2. **Publish directly** (`devbox run publish-release`) — same pipeline, then
   takes it live. Right for routine patch releases.
3. **Publish an existing draft** (`devbox run publish-release`) — if a draft is
   already sitting there, the command lists it and offers to finish it. Check
   with `devbox run release-status` first.

## 3. Write the title and notes

The script prompts for these in `$EDITOR` when a human drives it. When you're
driving, write them yourself and pass them as flags — that's the point of doing
this through an agent.

Get the raw material from GitHub's generated changelog:

```bash
gh api repos/jetify-com/devbox/releases/generate-notes \
  -f tag_name=<version> -f previous_tag_name=<prev> -f target_commitish=main --jq '.body'
```

That's a dump of PR titles. Rewrite it into user-facing notes in the house style
that 0.17.4 established and 0.18.0 is the cleanest example of:

````markdown
## What's Changed

### 💥 Breaking Changes

* **What was removed** — what users must do instead, by @author ([#1234](https://github.com/jetify-com/devbox/pull/1234))

### ✨ New Features

* **Short bold lead-in** — what changed and why a user cares, by @author ([#1234](...))

### 🐛 Bug Fixes

* Plain one-liners for small fixes, by @author ([#1234](...)).
* **Group related fixes** — combine several PRs into one bullet when they share
  a root cause ([#1234](...)) and ([#1235](...)), by @author.

### 🧹 Maintenance

* Dependency bumps, CI work, docs. Group aggressively; nobody reads this section
  line by line.

## New Contributors
* @newperson made their first contribution in https://github.com/jetify-com/devbox/pull/1234

**Full Changelog**: https://github.com/jetify-com/devbox/compare/<prev>...<version>
````

Rules that matter:

- **Lead with impact, not the commit subject.** "Shells in paths with spaces now
  work" beats "quote the shellrc source guard".
- **Breaking changes go first and say what to do instead.** For a release that
  removes features, this is the whole story — say plainly what's gone and what
  replaces it, or that nothing does.
- **Only include sections with content.** Drop the ones that are empty.
- **Keep attribution.** Every bullet ends with `by @author` and its PR link.
- **Use plain backticks for code.** Do *not* escape them as `` \` ``. Nothing
  re-interprets the body — goreleaser's Discord announcer is `enabled: false` in
  `.goreleaser.yaml` — and Markdown renders `` \` `` as a literal backtick
  rather than opening a code span. That's why 0.17.4's published notes are full
  of stray backslashes. The script unescapes any that slip through, but don't
  write them.

For the title, default to the bare version (`0.18.0`). A short theme suffix is
fine when the release has one: `0.18.0 — Devbox goes fully local`.

**Show the user the title and full notes and get explicit sign-off before
running anything.** That's the last checkpoint before it goes live.

## 4. Run it

Save the notes to `.context/release-notes-<version>.md`, then:

```bash
node scripts/release.ts --draft \
  --version 0.18.0 \
  --title "0.18.0" \
  --notes-file .context/release-notes-0.18.0.md \
  --yes
```

Swap `--draft` for `--publish` to go live. Notes on driving it as an agent:

- **`--yes` is required when you run it.** Its confirmation prompts need a TTY,
  and your shell doesn't have one — without `--yes` it stops with "this step
  needs an answer but stdin is not a terminal". Only pass it after the user has
  signed off; it's skipping *their* checkpoint, not adding one.
- **It blocks for ~15 minutes** waiting on `cli-release`. Run it in the
  background and check back, rather than letting the call time out.
- **It's resumable.** Every step is idempotent, so if it fails partway, fix the
  cause and re-run the same command — it picks up where it left off rather than
  duplicating work.
- **`--skip-cli-tests` exists but is a loaded gun.** It only skips the *check*
  that the latest `cli-tests` run on `main` is green; `cli-release` still runs
  the whole suite and fails the build if it isn't. Use it when the check is
  wrong (a run still in progress you've already inspected, a flake you've
  confirmed), not to push past a genuinely red `main`.

If `flake.nix` needs bumping, the script offers to do the whole thing: it
rewrites `lastTag`, refreshes `vendor-hash` and `flake.lock`, commits to
`bump-flake-<version>`, pushes, opens the PR, and puts you back on `main` with a
clean tree. Then it stops — that bump has to be reviewed and merged before the
tag is pushed, otherwise the tagged commit ships the wrong version string. Get
it merged and re-run the same command; preflight pulls the new `main` and
carries on. A re-run while the PR is still open stops immediately with its URL
rather than opening a second one.

Preflight is picky about the checkout on purpose, since the tag lands on
whatever `HEAD` is. A `main` that's merely behind `origin/main` and clean
fast-forwards itself; anything else (wrong branch, dirty tree, local-only
commits) stops with the command that fixes it.

## Why the order is what it is

Don't work around these; they're why the script exists.

- **Draft before tag.** goreleaser runs with `release.disable` and only builds
  `dist/`; `cli-release` uploads that to the release for the tag, looking it up
  by tag with `gh`. With no draft it creates one from GitHub's generated notes —
  usable, but not the notes you wrote. (goreleaser used to do the upload itself,
  but it matched drafts by *title*, so any release with a real title got a
  silent duplicate draft with bare-commit-SHA notes. That's what shipped as the
  0.17.5 release notes, and what stalled 0.18.0.)
- **Publish after the build.** `docker-image-release` fires on the release event
  and immediately downloads the release tarballs to bake into the image.
  Publishing before `cli-release` uploads them fails the Docker build. That's
  exactly what happened on 0.17.3 and 0.17.5, both published from the GitHub UI,
  which creates the tag and publishes in one action.
- **Flake bump before the `cli-tests` check.** The bump has to merge into `main`,
  which re-runs `cli-tests` there — so a result read before the bump is about a
  commit that won't be released. Settling the bump first also means a red `main`
  doesn't hide the fact that a bump PR is needed.
- **Publish from local credentials, not CI.** GitHub doesn't trigger workflows
  from events raised by `GITHUB_TOKEN`. That's why CI-created edge releases never
  trigger `docker-image-release`, and why publishing can't just be a CI step.

## If something is stuck

`devbox run release-status` reports the tag, draft, asset count, `cli-release`
result and the current `flake.nix` version in one shot.

| Symptom | Cause | Fix |
| --- | --- | --- |
| `cli-release` never started | Tag push didn't land | Re-run the same command |
| `cli-release` failed in `tests` | Red `main` (see below) | Fix the test, re-run |
| Draft has 0 assets | The upload step didn't run or failed | Check `cli-release`'s "Attach artifacts" step, re-run |
| Published but installer serves the old version | `cli-post-release` failed or is still running | `gh run list --workflow=cli-post-release.yml` |
| Docker build failed | Published before assets uploaded | Re-run `docker-image-release` via `workflow_dispatch` with the tag |

## Known breakage

Check these are still true before blaming the release itself:

- **`main` was red from 2026-07-02 until #2951.** The macOS `zig-hello-world`
  example test failed because `build.zig` used the pre-Zig-0.12 API while
  `devbox.lock` pinned zig 0.11.0; the upgrade to zig 0.16 fixed it. A red
  `main` blocks *all* releases — `cli-release` gates on the test suite — so
  check the current state rather than assuming either way.
- **`flake.nix` drifts.** It sat at `0.17.3` through both the 0.17.4 and 0.17.5
  releases. The script catches this now and opens the bump PR for you.
