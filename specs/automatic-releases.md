# Automatic Devbox CLI releases

Status: proposal

## Goal

Ship a stable devbox release every week with nobody involved, whenever `main`
has changed. Anything merged into `main` is considered ready to ship once it
passes tests. People are only needed when something fails.

## Where we are today

`scripts/release.ts` (wrapped by `devbox run draft-release` / `publish-release`
and the `release-devbox` skill) already does every mechanical step in the right
order. A person still has to:

1. run it locally and pick the version,
2. write the title and notes,
3. get the flake bump PR (`bump-flake-<version>`) reviewed and merged, then run it again,
4. watch `cli-release` for ~15 minutes, then publish.

It runs locally because **events raised with `GITHUB_TOKEN` do not trigger
other workflows**. A tag pushed by CI would not start `cli-release`, a PR
opened by CI would not get `cli-tests`, and a release published by CI would
not start `docker-image-release` or `cli-post-release`. The plan below gets
around this with a GitHub App token, not a personal token.

## One change to the requested order

The request says "create tag, then create the flake PR". That order would ship
the wrong version string. `flake.nix` hard-codes `lastTag`, and the tagged
commit has to contain the bumped value. `release.ts` already enforces this
("Flake bump before tag"). So the automated order is:

**bump PR → Greptile approves → merge → tag the merge commit → tests → build → publish.**

The version is still chosen at the start, when the PR opens, so "decide the
release" still happens first.

## Pipeline

```
Tue 16:17 UTC  release-weekly.yml (schedule)
   │  any shippable commits since last tag?  ── no ──▶ exit
   │  compute next version (release.ts --next-version --json)
   │  close any stale bump-flake-* PRs
   │  bump flake.nix lastTag, vendor-hash, flake.lock (needs nix)
   │  open PR bump-flake-<v> as the release App, label `auto-release`
   │  enable auto-merge (squash)
   ▼
PR: cli-tests (required) + Greptile review (required approval)
   │  both green ──▶ GitHub auto-merge squashes into main (as the App)
   ▼
release-tag.yml (pull_request: closed, merged, label auto-release)
   │  create draft release <v> with generated notes
   │  push tag <v> on merge_commit_sha (App token → triggers cli-release)
   ▼
cli-release.yml (push: tags)  ── existing ──
   │  tests (now with run-mac-tests: true)
   │  goreleaser build, attest, upload to the draft
   │  NEW publish job: draft → published, --latest   (App token)
   ▼
release: published / released  ── existing ──
   docker-image-release.yml, cli-post-release.yml
```

## Work items

### 1. Release GitHub App (one-time setup, admin)

- Create a GitHub App `devbox-release-bot` with these permissions: contents:
  write, pull-requests: write, workflows: read, checks: read. Install it on
  `jetify-com/devbox` only.
- Store `RELEASE_APP_ID` and `RELEASE_APP_PRIVATE_KEY` in the existing
  `release` environment.
- Each workflow mints a short-lived token with `actions/create-github-app-token`.
  Use this token for every push, PR, merge, tag and publish. That is what lets
  the downstream workflows fire.
- Tag ruleset: only the App, plus admins as a manual fallback, may create
  non-edge tags.

### 2. Make `release.ts` usable in CI (non-interactive)

Reuse its logic rather than re-implementing it in YAML:

- `--next-version --json`: prints `{ prev, version, bump, reason, commits, shippable }`.
  `shippable` is false when the only commits since the last tag are
  `chore(release): bump flake …` commits, so a quiet week doesn't release.
  Optional, decided by us: also ignore `docs:`- and `examples`-only changes.
- `--bump-flake --version <v> --no-pr`: rewrites `lastTag` and runs
  `update-hash` and `nix flake update`, without the git and PR work. The
  workflow does the commit and PR with the App token. Note: also running
  `nix flake update` makes every weekly release bump nixpkgs. Keep that (it is
  current behaviour) or drop it so a release bump only changes devbox.
- Keep the existing interactive flows as they are. Humans can still cut an
  out-of-band release, and the automation must tolerate that (see §7).

### 3. `release-weekly.yml` (new)

- Triggers: `schedule: '17 16 * * 2'` (Tuesday, off the hour, away from the
  Thursday edge build and the 08:30 nightly tests), plus `workflow_dispatch`
  with an optional `version` override and a `dry_run` flag.
- `concurrency: release-weekly`.
- Steps: checkout `fetch-depth: 0` → install Nix
  (`DeterminateSystems/nix-installer-action`) → install devbox →
  `release.ts --next-version --json` → exit early if not shippable → skip if
  tag `<v>` already exists → close any older open `auto-release` PRs with a
  "superseded by" comment → bump → commit `chore(release): bump flake lastTag to <v>`
  on `bump-flake-<v>` → push with the App token → `gh pr create --label auto-release`
  → `gh pr merge --auto --squash`.
- If Greptile only reviews when mentioned, also post `@greptileai review`.

### 4. Gate the merge on Greptile + tests

Preferred: **native auto-merge plus a ruleset on `main`**. The ruleset requires
1 approving review and the `cli-tests` checks. When Greptile submits an
approving review and the checks go green, GitHub merges by itself. No polling,
no extra workflow.

Things to check before building this:

- **Does Greptile submit an `APPROVED` review?** By default it posts a COMMENT
  review with a confidence score. If it can be set to approve, enable that. If
  it can't, add a small `release-approve.yml` on `pull_request_review` /
  `issue_comment` from `greptile-apps[bot]`, restricted to `auto-release` PRs
  authored by the App. When Greptile reports no blocking issues (for example,
  confidence ≥ 4/5), it merges with the App token. The App can't approve a PR
  it authored, so this workflow merges directly rather than approving.
- **Does Greptile review bot-authored PRs?** Some configurations skip them.
  Allow the App's author in Greptile's settings.
- Requiring 1 approval on `main` applies to human PRs as well. Confirm it
  matches the current branch protection.
- **Timeout:** if the PR hasn't merged after 48h, post to Slack
  (`SLACK_CLI_RELEASE_WEBHOOK_URL`). The next weekly run supersedes it anyway.

### 5. `release-tag.yml` (new)

- Trigger: `pull_request: types: [closed]`. Runs only when `merged == true`,
  the PR has the `auto-release` label, the head branch matches `bump-flake-*`,
  and the author is the App. All four conditions are required, so nobody can
  start a release by labelling their own PR.
- Read `<v>` from the merged `flake.nix` (`lastTag`) at `merge_commit_sha`.
  Don't use the branch name.
- **Notes:** create the draft *before* tagging, because `cli-release` attaches
  assets to an existing draft. Use `gh api …/releases/generate-notes` against
  the previous tag, so the notes include everything merged up to the merge
  commit, even work merged while the PR waited. Optional later step: run the
  existing Claude action (`claude.yml`) to rewrite the notes in the house style
  from the `release-devbox` skill, falling back to the generated notes if it
  fails. Notes are never a reason to block a release.
- Push the tag to `merge_commit_sha` with the App token. That triggers
  `cli-release`.

### 6. `cli-release.yml` changes

- `tests: uses: ./.github/workflows/cli-tests.yaml` with
  `with: { run-mac-tests: true }`. Today the release gate skips macOS: the
  `workflow_call` path excludes `macos-latest` unless that input is set. "All
  tests pass" should include macOS before we ship to macOS users without a
  human looking.
- New `publish` job after `release`, using `environment: release`:
  - Runs only when the tag push came from the App
    (`github.actor == 'devbox-release-bot[bot]'`). Humans running
    `draft-release` keep their draft. `release.ts --publish` already treats
    "already published" as success, so the two paths don't conflict.
  - Use the same checks as `stepPublish`: the release has assets and
    `checksums.txt`. Then
    `gh release edit <v> --draft=false --latest` (`--latest=false` for
    prereleases), using the App token so `docker-image-release` and
    `cli-post-release` fire.
  - Add a smoke check before publishing. Download the linux/amd64 archive from
    the draft and assert that `devbox version` equals the tag. This catches a
    stale `lastTag` or broken ldflags (the child audit below expands on this).
- Slack notification: report each stage (test failure, build failure,
  published).

### 7. Failure modes and how they resolve

| Situation | Behaviour |
| --- | --- |
| No shippable commits this week | Weekly job exits; no PR. |
| Greptile requests changes / never approves | PR stays open; Slack at 48h; next week's run closes it and opens a new bump PR. |
| `cli-tests` red on the bump PR | Auto-merge never fires; same as above. Fix main and dispatch `release-weekly` manually. |
| Tests fail in `cli-release` after tagging | Tag + draft exist, nothing published. Fix forward on main, then let the next weekly run cut `<v+1>`. Delete the orphan draft (tag stays; versions are never reused). Or re-run `cli-release` if it was a flake. |
| Human cut a release mid-week | Next run computes from the new latest tag; `release.ts` still works as before. |
| Tag already exists for computed version | Weekly job skips and alerts rather than overwriting. |
| Docker / S3 post-release fails | Same as today (non-blocking); runbook in the skill applies. |

### 8. Docs

- Update `.agents/skills/release-devbox/SKILL.md`: releases are automatic; the
  skill becomes the manual override and the stuck-release runbook. Add the new
  workflows to its "If something is stuck" table.
- Remove the "Publish from local credentials, not CI" rationale; the App token
  replaces it.

## Rollout

1. Ship `release.ts` flags, the App, and the workflows with `dry_run` on.
   The weekly job opens the PR but `release-tag` only logs what it would do.
2. Run one release end to end via `workflow_dispatch`, watched by a human.
3. Turn on the schedule. Review the first two automatic releases afterwards.

## Test confidence (in progress)

Since tests become the only release gate, a separate session is auditing
coverage and adding fast smoke tests on branch
`claude/devbox-test-coverage-audit`. Planned smoke tests: run the built binary
for `version` / `init` / `add` / `run`, check that `nix build` reports the same
version as the tag, and run install.sh against the built archive. All of these
run in parallel with existing jobs so CI doesn't get slower.

## Open questions

1. Can Greptile be set to submit `APPROVED` reviews, and does it review
   App-authored PRs? This decides between §4's two options.
2. Should `docs:`/`examples`-only weeks skip the release?
3. Keep `nix flake update` (nixpkgs bump) in every release PR?
4. Day and time for the weekly cut (proposed Tue 16:17 UTC).
