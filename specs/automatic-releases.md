# Automatic Devbox CLI releases

Status: implemented (needs the one-time setup in §1)

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
Tue 08:17 PST  release-weekly.yml (schedule)
   │  any shippable commits since last tag?  ── no ──▶ exit
   │  compute next version (release.ts --next-version --json)
   │  close any stale bump-flake-* PRs
   │  bump flake.nix lastTag, vendor-hash, flake.lock (needs nix)
   │  open PR bump-flake-<v> as the release App, label `auto-release`
   │  enable auto-merge (squash)
   ▼
PR: cli-tests (required) + Greptile approval (required)
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

- Create a GitHub App (e.g. `devbox-release-bot`) with these repository
  permissions: contents: write, pull-requests: write, issues: write (to create
  the `auto-release` label), checks: read. Install it on `jetify-com/devbox`
  only.
- In the existing `release` environment, store the variables `RELEASE_APP_ID`
  and `RELEASE_APP_SLUG` (the App's slug, e.g. `devbox-release-bot`) and the
  secret `RELEASE_APP_PRIVATE_KEY`. `RELEASE_APP_SLUG` can also be a repo-level
  variable. `release-approve.yml` reads it without the environment.
- Repo settings: turn on "Allow auto-merge", and turn on "Allow GitHub Actions
  to create and approve pull requests" (only the §4 fallback uses that).
- Each workflow mints a short-lived token with `actions/create-github-app-token`.
  Use this token for every push, PR, merge, tag and publish. That is what lets
  the downstream workflows fire.
- Tag ruleset: only the App, plus admins as a manual fallback, may create
  non-edge tags.

### 2. Make `release.ts` usable in CI (non-interactive)

Reuse its logic rather than re-implementing it in YAML:

- `--next-version`: prints JSON `{ prev, version, bump, reason, commits, shippable }`.
  `shippable` is false when every commit since the last tag is one of:
  a `chore(release): bump flake …` commit, a `docs:` commit, or a commit that
  only touches `examples/` or docs paths. A week with only those changes
  doesn't release; they ship with the next real release.
- `--bump-flake --version <v> --no-pr`: rewrites `lastTag` and runs
  `update-hash` and `nix flake update`, without the git and PR work. The
  workflow does the commit and PR with the App token. Running
  `nix flake update` here bumps nixpkgs with every release, as releases do
  today (decided: keep it).
- Keep the existing interactive flows as they are. Humans can still cut an
  out-of-band release, and the automation must tolerate that (see §7).

### 3. `release-weekly.yml` (new)

- Triggers: `schedule: '17 16 * * 2'`, which is Tuesday 08:17 PST. It's off
  the hour and away from the 08:30 UTC nightly tests. GitHub cron is UTC, so it runs at 09:17 during PDT. Plus `workflow_dispatch`
  with an optional `version` override.
- `concurrency: release-weekly`.
- Steps: checkout `fetch-depth: 0` → Node 24 →
  `release.ts --next-version` → exit early if not shippable → skip if
  tag `<v>` already exists → close any older open `auto-release` PRs with a
  "superseded by" comment → bump → commit `chore(release): bump flake lastTag to <v>`
  on `bump-flake-<v>` → push with the App token → `gh pr create --label auto-release`
  → `gh pr merge --auto --squash`.
- If Greptile only reviews when mentioned, also post `@greptileai review`.

### 4. Gate the merge on Greptile + tests

Greptile submits an `APPROVED` review on its own when it finds no blocking
issues, so the gate is just **native auto-merge plus a ruleset on `main`**:

- The ruleset requires 1 approving review and the `cli-tests` checks.
- The weekly job enables auto-merge (`gh pr merge --auto --squash`) when it
  opens the PR. Once Greptile approves and the checks are green, GitHub merges
  it. No polling and no extra workflow. The merge is attributed to the App,
  since it enabled auto-merge, so the resulting push to `main` triggers
  workflows normally.
- If Greptile requests changes or never approves, the PR just stays open.

**Fallback if Greptile only comments instead of approving:** approve in CI.
Add a small `release-approve.yml` that runs on `pull_request_review` and
`issue_comment` events from `greptile-apps[bot]`. It is limited to
`auto-release` PRs authored by the App. When Greptile's verdict passes
(confidence ≥ 4/5 and no P0/P1 findings) and the reviewed commit is still the
PR head, it runs `gh pr review --approve` with `GITHUB_TOKEN`. That needs the
repo setting "Allow GitHub Actions to create and approve pull requests". The
`github-actions[bot]` approval counts toward the ruleset, because the PR's
author is the App, not Actions. Auto-merge then fires as above, and
`cli-tests` is still required, so no bypass rules are needed.

Things to confirm when setting this up:

- **Greptile reviews App-authored PRs.** Some configurations skip bot authors.
  Allow the App in Greptile's settings.
- **The ruleset matches `main`'s current protection.** Requiring 1 approval
  applies to human PRs too.
- **Timeout:** if the PR hasn't merged after 48h, post to Slack
  (`SLACK_CLI_RELEASE_WEBHOOK_URL`). The next weekly run supersedes it anyway.

### 5. `release-tag.yml` (new)

- Trigger: the merge's `push` to `main` (paths: `flake.nix`), not
  `pull_request: closed`. A `pull_request` run's ref is `refs/pull/N/merge`,
  which the `release` environment's branch rules may not allow. The push comes
  from the App, which enabled auto-merge, so it triggers workflows.
- Looks up the PR for the pushed commit, and continues only when it merged, it
  has the `auto-release` label, its head branch matches `bump-flake-*`, and its
  author is the App. All four conditions are required, so nobody can start a
  release by labelling their own PR.
- Read `<v>` from the merged `flake.nix` (`lastTag`), and check that it matches
  the branch name.
- **Notes:** create the draft *before* tagging, because `cli-release` attaches
  assets to an existing draft. Use `gh api …/releases/generate-notes` against
  the previous tag, so the notes include everything merged up to the merge
  commit, even work merged while the PR waited. Optional later step: run the
  existing Claude action (`claude.yml`) to rewrite the notes in the house style
  from the `release-devbox` skill, falling back to the generated notes if it
  fails. Notes are never a reason to block a release.
- Push an annotated tag on the merge commit with the App token. That triggers
  `cli-release`.

### 6. `cli-release.yml` changes

- `tests: uses: ./.github/workflows/cli-tests.yaml` with
  `run-mac-tests: true` for tag pushes. Today the release gate skips macOS: the
  `workflow_call` path excludes `macos-latest` unless that input is set. "All
  tests pass" should include macOS before we ship to macOS users without a
  human looking.
- New `publish` job after `release`, using `environment: release`:
  - Runs only when the tag push came from the App
    (`github.actor == '<RELEASE_APP_SLUG>[bot]'`). Humans running
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
    Release archives are built with `build.Launcher=true`. Run outside the
    launcher's install directory on a clean runner, the binary looks up the
    latest *published* release and runs that (`internal/launcher/launcher.go`,
    `selectVersion`), so a plain `devbox version` would check the previous
    release. Run it as `DEVBOX_USE_VERSION=<v> ./devbox version`. If the binary
    really is `<v>`, the launcher runs it directly. If it reports another
    version, the launcher tries to download `<v>`, which isn't published yet,
    so the check fails.
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

No dry run. Once the App, variables and repo settings from §1 are in place:

1. Start the first release with `workflow_dispatch` on `release-weekly`, and
   watch it end to end.
2. After that the Tuesday schedule takes over. Review the first couple of
   automatic releases afterwards.

## Test confidence (in progress)

Since tests become the only release gate, a separate session is auditing
coverage and adding fast smoke tests on branch
`claude/devbox-test-coverage-audit`. Planned smoke tests: run the built binary
for `version` / `init` / `add` / `run`, check that `nix build` reports the same
version as the tag, and run install.sh against the built archive. All of these
run in parallel with existing jobs so CI doesn't get slower.

## Decisions

1. Weeks with only `docs:` or `examples/` changes skip the release.
2. Each release PR also runs `nix flake update`, so nixpkgs is bumped every
   release.
3. The weekly run is Tuesday 08:17 PST (16:17 UTC).
4. Greptile approves automatically. If it ever stops, the CI approval fallback
   in §4 takes over.
