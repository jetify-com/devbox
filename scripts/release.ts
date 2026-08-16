#!/usr/bin/env node
//
// release.ts drives a Devbox CLI release.
//
// There are two entry points, both of which walk every step in order:
//
//   devbox run draft-release     # build the release, leave it as a draft
//   devbox run publish-release   # build the release and take it live
//
// publish-release can start from scratch or finish a draft that draft-release
// (or a previous, interrupted run) already left behind.
//
// You supply the tag, title and description yourself — this script never
// invents them. Everything else is mechanical and happens in a fixed order,
// because that order is load-bearing:
//
//   1. The draft must exist before the tag is pushed. goreleaser runs with
//      `use_existing_draft` and attaches its artifacts to whatever draft it
//      finds, keeping the body. With no draft it creates one whose body is a
//      bare list of commit SHAs — that is what shipped as the 0.17.5 notes.
//
//   2. The release must not be published until the build has uploaded its
//      artifacts. `docker-image-release` fires on the release event and
//      immediately downloads the tarballs to bake into the image; publishing
//      early fails the Docker build, as happened on 0.17.3 and 0.17.5.
//
//   3. Publishing happens here, from your own credentials, not from CI. GitHub
//      does not trigger workflows from events raised by GITHUB_TOKEN, so a
//      release published by a workflow would never reach cli-post-release or
//      docker-image-release.
//
// Every step is idempotent, so re-running after a failure partway through
// resumes rather than duplicating work.

import { spawnSync } from "node:child_process";
import { createInterface, type Interface } from "node:readline/promises";
import fs from "node:fs";
import os from "node:os";
import path from "node:path";

const REPO = "jetify-com/devbox";
const MAIN_BRANCH = "main";
const ROOT = path.resolve(import.meta.dirname, "..");

// ---------------------------------------------------------------------------
// Output
// ---------------------------------------------------------------------------

const useColor = process.stdout.isTTY && !process.env.NO_COLOR;
const color = (code: string, s: string) => (useColor ? `\x1b[${code}m${s}\x1b[0m` : s);
const bold = (s: string) => color("1", s);
const dim = (s: string) => color("2", s);
const green = (s: string) => color("32", s);
const yellow = (s: string) => color("33", s);
const red = (s: string) => color("31", s);

let stepNumber = 0;
let totalSteps = 0;

function step(title: string): void {
  stepNumber++;
  console.log(`\n${bold(`[${stepNumber}/${totalSteps}] ${title}`)}`);
}

const info = (msg: string) => console.log(`  ${msg}`);
const ok = (msg: string) => console.log(`  ${green("✓")} ${msg}`);
const warn = (msg: string) => console.log(`  ${yellow("!")} ${msg}`);

class ReleaseError extends Error {}

function fail(msg: string): never {
  throw new ReleaseError(msg);
}

// ---------------------------------------------------------------------------
// Process helpers
// ---------------------------------------------------------------------------

type RunResult = { status: number; stdout: string; stderr: string };

function run(cmd: string, args: string[], opts: { cwd?: string } = {}): RunResult {
  const res = spawnSync(cmd, args, {
    cwd: opts.cwd ?? ROOT,
    encoding: "utf8",
    maxBuffer: 64 * 1024 * 1024,
  });
  if (res.error) fail(`failed to run ${cmd}: ${res.error.message}`);
  return {
    status: res.status ?? 1,
    stdout: (res.stdout ?? "").trim(),
    stderr: (res.stderr ?? "").trim(),
  };
}

// Runs a command and fails the release if it exits non-zero.
function capture(cmd: string, args: string[]): string {
  const res = run(cmd, args);
  if (res.status !== 0) {
    fail(`${cmd} ${args.join(" ")} failed:\n${res.stderr || res.stdout}`);
  }
  return res.stdout;
}

// Runs a command with its output streamed straight to the terminal, for things
// long enough that the user needs to see progress (builds, nix, gh run watch).
function stream(cmd: string, args: string[]): number {
  const res = spawnSync(cmd, args, { cwd: ROOT, stdio: "inherit" });
  if (res.error) fail(`failed to run ${cmd}: ${res.error.message}`);
  return res.status ?? 1;
}

const git = (...args: string[]) => capture("git", args);
const gitStatus = (...args: string[]) => run("git", args).status;

function gh(...args: string[]): string {
  return capture("gh", args);
}

function ghJSON<T>(...args: string[]): T {
  return JSON.parse(gh(...args)) as T;
}

function requireCommand(cmd: string): void {
  // `which`, not `command -v`: the latter is a shell builtin and spawnSync
  // would fail to exec it.
  const res = spawnSync("which", [cmd], { encoding: "utf8" });
  if (res.error || res.status !== 0) fail(`'${cmd}' is required but not installed`);
}

// ---------------------------------------------------------------------------
// Prompts
// ---------------------------------------------------------------------------

let rl: Interface | null = null;

function reader(): Interface {
  if (!process.stdin.isTTY) {
    fail("this step needs an answer but stdin is not a terminal — pass the value as a flag instead");
  }
  rl ??= createInterface({ input: process.stdin, output: process.stdout });
  return rl;
}

function closeReader(): void {
  rl?.close();
  rl = null;
}

async function ask(question: string, fallback?: string): Promise<string> {
  const suffix = fallback ? dim(` [${fallback}]`) : "";
  const answer = (await reader().question(`  ${question}${suffix}: `)).trim();
  if (answer) return answer;
  if (fallback !== undefined) return fallback;
  return ask(question, fallback);
}

async function confirm(question: string, fallback = true): Promise<boolean> {
  const hint = fallback ? "Y/n" : "y/N";
  const answer = (await reader().question(`  ${question} ${dim(`[${hint}]`)} `)).trim().toLowerCase();
  if (!answer) return fallback;
  return answer === "y" || answer === "yes";
}

async function choose<T extends string>(
  question: string,
  options: { value: T; label: string }[],
): Promise<T> {
  console.log(`  ${question}`);
  options.forEach((opt, i) => console.log(`    ${bold(String(i + 1))}. ${opt.label}`));
  for (;;) {
    const answer = (await reader().question(`  Choice ${dim(`[1-${options.length}]`)}: `)).trim();
    const index = Number.parseInt(answer, 10);
    if (index >= 1 && index <= options.length) return options[index - 1]!.value;
    console.log(`  ${red("!")} Enter a number between 1 and ${options.length}.`);
  }
}

// Opens $EDITOR on a scratch file so the user can write the release notes, then
// reads the result back. Lines starting with "#" are stripped so we can put
// guidance in the template without it ending up in the notes.
async function editText(template: string, filename: string): Promise<string> {
  const editor = process.env.VISUAL || process.env.EDITOR;
  if (!editor) {
    fail("no $EDITOR set — set one, or pass the notes with --notes-file <path>");
  }
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "devbox-release-"));
  const file = path.join(dir, filename);
  fs.writeFileSync(file, template);

  console.log(`  Opening ${dim(editor)} — save and quit when you're done.`);
  const status = spawnSync(editor, [file], { stdio: "inherit", shell: true }).status;
  if (status !== 0) fail(`editor exited with status ${status}`);

  const body = fs
    .readFileSync(file, "utf8")
    .split("\n")
    .filter((line) => !line.startsWith("#"))
    .join("\n")
    .trim();
  fs.rmSync(dir, { recursive: true, force: true });
  if (!body) fail("release notes were empty — aborting");
  return body;
}

// ---------------------------------------------------------------------------
// Version helpers
// ---------------------------------------------------------------------------

// Devbox tags are bare semver with no leading "v" (0.17.5, 0.18.0-dev).
const VERSION_RE = /^(\d+)\.(\d+)\.(\d+)(?:-([0-9A-Za-z.-]+))?$/;

type Version = { major: number; minor: number; patch: number; prerelease?: string };

function parseVersion(raw: string): Version | null {
  const m = VERSION_RE.exec(raw);
  if (!m) return null;
  return {
    major: Number(m[1]),
    minor: Number(m[2]),
    patch: Number(m[3]),
    prerelease: m[4],
  };
}

function validateVersion(raw: string): string {
  if (raw.startsWith("v")) {
    fail(`drop the leading "v": devbox tags are bare semver (${raw.slice(1)}, not ${raw})`);
  }
  if (!parseVersion(raw)) {
    fail(`"${raw}" is not a bare semver version (expected e.g. 0.18.0)`);
  }
  return raw;
}

const isPrerelease = (version: string) => version.includes("-");

// Edge tags (0.0.0-edge.*) are weekly snapshots, not release points, so they
// never count as "the previous release".
const isEdge = (tag: string) => tag.includes("-edge.");

function previousTag(exclude?: string): string {
  const tags = git("tag", "--list", "--merged", "HEAD", "--sort=-creatordate").split("\n");
  for (const tag of tags) {
    const t = tag.trim();
    if (!t || isEdge(t) || t === exclude) continue;
    return t;
  }
  fail("could not determine the previous release tag");
}

function flakeLastTag(): string {
  const flake = fs.readFileSync(path.join(ROOT, "flake.nix"), "utf8");
  const m = /lastTag = "([^"]*)"/.exec(flake);
  if (!m) fail("could not find lastTag in flake.nix");
  return m[1]!;
}

// ---------------------------------------------------------------------------
// Change analysis
// ---------------------------------------------------------------------------

type Commit = { subject: string; body: string; author: string; breaking: boolean; type: string };

function commitsSince(tag: string): Commit[] {
  // \x1e separates commits, \x1f separates fields — neither shows up in commit
  // messages, unlike the newlines a commit body is full of.
  const raw = git("log", "--no-merges", `--format=%s%x1f%b%x1f%an%x1e`, `${tag}..HEAD`);
  if (!raw) return [];
  return raw
    .split("\x1e")
    .map((chunk) => chunk.trim())
    .filter(Boolean)
    .map((chunk) => {
      const [subject = "", body = "", author = ""] = chunk.split("\x1f");
      const conventional = /^(\w+)(\([^)]*\))?(!)?:/.exec(subject);
      return {
        subject,
        body,
        author,
        type: conventional?.[1] ?? "other",
        breaking: Boolean(conventional?.[3]) || body.includes("BREAKING CHANGE"),
      };
    });
}

type Recommendation = { version: string; bump: "minor" | "patch"; reason: string };

// Devbox is pre-1.0, so the convention in its history is: breaking changes take
// a minor bump (0.17.5 -> 0.18.0), everything else takes a patch bump — 0.17.4
// shipped new features as a patch.
function recommendVersion(prev: string, commits: Commit[]): Recommendation {
  const version = parseVersion(prev);
  if (!version) fail(`previous tag "${prev}" is not parseable semver`);

  const breaking = commits.filter((c) => c.breaking);
  if (breaking.length > 0) {
    return {
      version: `${version.major}.${version.minor + 1}.0`,
      bump: "minor",
      reason: `${breaking.length} breaking change${breaking.length === 1 ? "" : "s"} since ${prev}`,
    };
  }
  return {
    version: `${version.major}.${version.minor}.${version.patch + 1}`,
    bump: "patch",
    reason: `no breaking changes since ${prev}`,
  };
}

function summarizeChanges(prev: string, commits: Commit[]): void {
  info(`${bold(String(commits.length))} commits since ${bold(prev)}:`);
  const breaking = commits.filter((c) => c.breaking);
  const features = commits.filter((c) => !c.breaking && c.type === "feat");
  const fixes = commits.filter((c) => !c.breaking && c.type === "fix");
  const rest = commits.filter((c) => !c.breaking && c.type !== "feat" && c.type !== "fix");

  const show = (label: string, list: Commit[]) => {
    if (list.length === 0) return;
    console.log(`\n  ${bold(label)} (${list.length})`);
    for (const c of list) console.log(`    • ${c.subject} ${dim(`— ${c.author}`)}`);
  };
  show("Breaking", breaking);
  show("Features", features);
  show("Fixes", fixes);
  show("Other", rest);
}

// ---------------------------------------------------------------------------
// Release state
// ---------------------------------------------------------------------------

type ReleaseInfo = {
  tagName: string;
  isDraft: boolean;
  isPrerelease: boolean;
  assetCount: number;
  url: string;
};

function getRelease(version: string): ReleaseInfo | null {
  const res = run("gh", [
    "release", "view", version, "--repo", REPO,
    "--json", "tagName,isDraft,isPrerelease,assets,url",
  ]);
  if (res.status !== 0) return null;
  const parsed = JSON.parse(res.stdout) as {
    tagName: string; isDraft: boolean; isPrerelease: boolean; assets: unknown[]; url: string;
  };
  return {
    tagName: parsed.tagName,
    isDraft: parsed.isDraft,
    isPrerelease: parsed.isPrerelease,
    assetCount: parsed.assets.length,
    url: parsed.url,
  };
}

function listDrafts(): { tagName: string; createdAt: string }[] {
  return ghJSON<{ tagName: string; createdAt: string; isDraft: boolean }[]>(
    "release", "list", "--repo", REPO, "--limit", "30",
    "--json", "tagName,createdAt,isDraft",
  ).filter((r) => r.isDraft);
}

const tagPushed = (version: string) =>
  gitStatus("ls-remote", "--exit-code", "--tags", "origin", version) === 0;

// ---------------------------------------------------------------------------
// Steps
// ---------------------------------------------------------------------------

function stepPreflight(opts: { requireCleanMain: boolean; checkMainCI: boolean }): void {
  step("Preflight");
  requireCommand("git");
  requireCommand("gh");
  if (run("gh", ["auth", "status"]).status !== 0) {
    fail("gh is not authenticated — run 'gh auth login'");
  }

  // Checks report in the order they run so the output reads top to bottom, but
  // all of them run before we bail, so one command surfaces every problem.
  let failures = 0;
  const check = (passed: boolean, okMsg: string, failMsg: string) => {
    if (passed) {
      ok(okMsg);
    } else {
      console.log(`  ${red("✗")} ${failMsg}`);
      failures++;
    }
  };

  if (opts.requireCleanMain) {
    const branch = git("rev-parse", "--abbrev-ref", "HEAD");
    check(branch === MAIN_BRANCH, `on ${MAIN_BRANCH}`, `not on ${MAIN_BRANCH} (currently on '${branch}')`);

    const dirty = gitStatus("diff", "--quiet") !== 0 || gitStatus("diff", "--cached", "--quiet") !== 0;
    check(!dirty, "working tree clean", "working tree is dirty — commit or stash your changes");
  }

  // --force on tags because a handful of old devbox tags were rewritten
  // upstream; without it every fetch fails with "would clobber existing tag".
  run("git", ["fetch", "--quiet", "origin", MAIN_BRANCH]);
  run("git", ["fetch", "--quiet", "--force", "--tags", "origin"]);

  if (opts.requireCleanMain) {
    const local = git("rev-parse", "HEAD");
    const remote = git("rev-parse", `origin/${MAIN_BRANCH}`);
    check(
      local === remote,
      `up to date with origin/${MAIN_BRANCH}`,
      `HEAD (${local.slice(0, 8)}) differs from origin/${MAIN_BRANCH} (${remote.slice(0, 8)})`,
    );
  }

  // cli-release gates the build on the full test suite, so a red main means the
  // tag push produces no artifacts at all. Only relevant when a build still has
  // to happen — a draft that already has its artifacts doesn't care what main
  // has done since.
  if (opts.checkMainCI) {
    const runs = ghJSON<{ conclusion: string | null }[]>(
      "run", "list", "--repo", REPO, "--workflow=cli-tests.yaml",
      "--branch", MAIN_BRANCH, "--limit", "1", "--json", "conclusion",
    );
    const conclusion = runs[0]?.conclusion ?? "none";
    check(
      conclusion === "success",
      `latest cli-tests on ${MAIN_BRANCH} is green`,
      `latest cli-tests on ${MAIN_BRANCH} is '${conclusion}' — cli-release will refuse to build`,
    );
  }

  if (failures > 0) fail("preflight failed — fix the items above before releasing");
}

async function stepChooseVersion(preset?: string): Promise<string> {
  step("Choose the version");
  const prev = previousTag();
  const commits = commitsSince(prev);
  if (commits.length === 0) fail(`no commits since ${prev} — nothing to release`);

  summarizeChanges(prev, commits);

  const rec = recommendVersion(prev, commits);
  console.log();
  info(`Recommended: ${bold(green(rec.version))} (${rec.bump} bump — ${rec.reason})`);

  if (preset) {
    const version = validateVersion(preset);
    ok(`Using ${bold(version)} (from --version)`);
    return version;
  }

  for (;;) {
    const answer = await ask("Version to release", rec.version);
    const version = answer.startsWith("v") ? answer.slice(1) : answer;
    if (!parseVersion(version)) {
      console.log(`  ${red("!")} "${answer}" is not bare semver (expected e.g. 0.18.0).`);
      continue;
    }
    if (gitStatus("rev-parse", "--verify", "--quiet", `refs/tags/${version}`) === 0) {
      console.log(`  ${red("!")} Tag ${version} already exists.`);
      continue;
    }
    return version;
  }
}

async function stepFlakeBump(version: string, autoYes: boolean): Promise<void> {
  step("Sync flake.nix");
  const current = flakeLastTag();
  if (current === version) {
    ok(`flake.nix lastTag is already ${version}`);
    return;
  }

  // flake.nix pins its own version string and nothing syncs it to the git tag,
  // so `nix build` reports a stale version unless it's bumped first. It sat at
  // 0.17.3 through both the 0.17.4 and 0.17.5 releases.
  warn(`flake.nix lastTag is ${current}, needs to be ${version}`);
  info("This has to land on main before the tag is pushed, or the tagged commit");
  info("ships the wrong version string.");

  if (!autoYes && !(await confirm(`Update flake.nix, vendor-hash and flake.lock now?`))) {
    fail(`flake.nix is out of date — bump lastTag to ${version} and merge before releasing`);
  }

  const flakePath = path.join(ROOT, "flake.nix");
  const flake = fs.readFileSync(flakePath, "utf8");
  const updated = flake.replace(`lastTag = "${current}";`, `lastTag = "${version}";`);
  if (updated === flake) fail("failed to rewrite lastTag in flake.nix");
  fs.writeFileSync(flakePath, updated);
  ok(`flake.nix lastTag ${current} -> ${version}`);

  info("Refreshing vendor-hash (devbox run update-hash)...");
  if (stream("devbox", ["run", "update-hash"]) !== 0) fail("devbox run update-hash failed");
  ok("vendor-hash refreshed");

  info("Updating flake.lock...");
  if (stream("nix", ["flake", "update"]) !== 0) warn("nix flake update failed — leaving flake.lock as is");

  console.log();
  console.log(git("--no-pager", "diff", "--stat", "--", "flake.nix", "flake.lock", "vendor-hash"));
  console.log();
  fail(
    `flake.nix is now updated locally but not merged. Open a PR, merge it, then re-run:\n` +
    `    git switch -c bump-flake-${version}\n` +
    `    git commit -am 'chore(release): bump flake lastTag to ${version}'\n` +
    `    gh pr create --base ${MAIN_BRANCH} --fill`,
  );
}

async function stepTitle(version: string, preset?: string): Promise<string> {
  step("Release title");
  if (preset) {
    ok(`Using "${preset}" (from --title)`);
    return preset;
  }
  info(`Convention is the bare version. A short theme suffix is fine for notable releases,`);
  info(`e.g. "${version} — Devbox goes fully local".`);
  return ask("Title", version);
}

async function stepNotes(version: string, notesFile?: string): Promise<string> {
  step("Release notes");
  if (notesFile) {
    const body = fs.readFileSync(notesFile, "utf8").trim();
    if (!body) fail(`${notesFile} is empty`);
    ok(`Using notes from ${notesFile} (${body.split("\n").length} lines)`);
    return body;
  }

  const prev = previousTag(version);
  info(`Fetching GitHub's generated changelog for ${prev}..${version} as a starting point...`);

  // GitHub's generator gives PR titles, authors, numbers, first-time
  // contributors and the compare link. It reads like a changelog dump, so it's
  // seeded into the editor as raw material rather than used as-is.
  const generated = ghJSON<{ body: string }>(
    "api", `repos/${REPO}/releases/generate-notes`,
    "-f", `tag_name=${version}`,
    "-f", `previous_tag_name=${prev}`,
    "-f", `target_commitish=${MAIN_BRANCH}`,
  ).body;

  const template = [
    `# Write the release notes for ${version}. Lines starting with "#" are ignored.`,
    `#`,
    `# House style (see 0.17.4 for a good example):`,
    `#   ## What's Changed`,
    `#   ### 💥 Breaking Changes   <- first, and say what to do instead`,
    `#   ### ✨ New Features`,
    `#   ### 🐛 Bug Fixes`,
    `#   ### 🧹 Maintenance        <- group aggressively`,
    `#`,
    `#   * **Bold lead-in** — impact, not the commit subject, by @author ([#1234](url))`,
    `#`,
    `# Drop sections that have no content. Escape backticks as \\\` — goreleaser`,
    `# interpolates this body into the Discord announcement template.`,
    `#`,
    `# Below is GitHub's auto-generated changelog. Rewrite it; don't ship it as is.`,
    ``,
    generated,
  ].join("\n");

  return editText(template, `release-notes-${version}.md`);
}

async function stepConfirm(version: string, title: string, notes: string, mode: Mode, autoYes: boolean): Promise<void> {
  step("Review");
  const prev = previousTag(version);
  console.log(`  ${dim("─".repeat(70))}`);
  console.log(`  ${bold("Version")}    ${version}${isPrerelease(version) ? dim(" (prerelease)") : ""}`);
  console.log(`  ${bold("Title")}      ${title}`);
  console.log(`  ${bold("Previous")}   ${prev}`);
  console.log(`  ${bold("Commit")}     ${git("rev-parse", "--short", "HEAD")}`);
  console.log(`  ${bold("Outcome")}    ${mode === "publish" ? red("published live") : "draft only"}`);
  console.log(`  ${dim("─".repeat(70))}`);
  for (const line of notes.split("\n")) console.log(`  ${dim("│")} ${line}`);
  console.log(`  ${dim("─".repeat(70))}`);

  if (autoYes) return;
  if (!(await confirm(mode === "publish" ? "Build and publish this release?" : "Create this draft?"))) {
    fail("aborted");
  }
}

function stepDraft(version: string, title: string, notes: string): void {
  step("Create the draft");
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), "devbox-release-"));
  const notesFile = path.join(dir, "notes.md");
  fs.writeFileSync(notesFile, notes);

  try {
    const prereleaseFlag = isPrerelease(version) ? ["--prerelease"] : [];
    const existing = getRelease(version);
    if (existing && !existing.isDraft) {
      // Re-running against a version that already shipped would silently
      // rewrite the notes of a live release.
      fail(`${version} is already published (${existing.url}) — pick a new version`);
    }
    if (existing) {
      info(`Draft for ${version} already exists — updating title and notes`);
      gh("release", "edit", version, "--repo", REPO,
        "--title", title, "--notes-file", notesFile, ...prereleaseFlag);
    } else {
      // A draft can name a tag that doesn't exist yet; GitHub only creates the
      // tag when the draft is published. We push the tag ourselves next so that
      // cli-release fires on a real push event.
      gh("release", "create", version, "--repo", REPO, "--draft",
        "--target", MAIN_BRANCH,
        "--title", title, "--notes-file", notesFile, ...prereleaseFlag);
    }
  } finally {
    fs.rmSync(dir, { recursive: true, force: true });
  }

  const release = getRelease(version);
  ok(`Draft ready: ${release?.url ?? version}`);
}

function stepTag(version: string): void {
  step("Push the tag");
  if (!getRelease(version)) fail(`no draft release for ${version} — the draft step must run first`);

  if (gitStatus("rev-parse", "--verify", "--quiet", `refs/tags/${version}`) !== 0) {
    git("tag", "-a", version, "-m", `Release ${version}`);
    ok(`Tagged ${git("rev-parse", "--short", "HEAD")} as ${version}`);
  } else {
    ok(`Tag ${version} already exists locally`);
  }

  if (tagPushed(version)) {
    ok(`Tag ${version} already pushed`);
    return;
  }
  git("push", "origin", `refs/tags/${version}`);
  ok(`Pushed ${version} — cli-release is now building`);
}

async function stepWait(version: string): Promise<void> {
  step("Wait for cli-release");
  const timeoutMs = Number(process.env.RELEASE_WAIT_TIMEOUT_MS ?? 60 * 60 * 1000);
  const deadline = Date.now() + timeoutMs;

  // A tag push shows up with the tag name as the run's head branch.
  let runId = "";
  info("Looking for the cli-release run...");
  while (!runId) {
    const runs = ghJSON<{ databaseId: number }[]>(
      "run", "list", "--repo", REPO, "--workflow=cli-release.yml",
      "--branch", version, "--event", "push", "--limit", "1", "--json", "databaseId",
    );
    if (runs[0]) {
      runId = String(runs[0].databaseId);
      break;
    }
    if (Date.now() > deadline) fail(`timed out waiting for a cli-release run for ${version}`);
    await new Promise((r) => setTimeout(r, 10_000));
  }

  info(`Watching run ${runId} — this takes ~15 minutes.`);
  if (stream("gh", ["run", "watch", runId, "--repo", REPO, "--exit-status"]) !== 0) {
    fail(
      `cli-release failed for ${version}. Inspect it with:\n` +
      `    gh run view ${runId} --repo ${REPO} --log-failed\n` +
      `Then re-run this command — it will resume from here.`,
    );
  }
  ok("cli-release succeeded");

  // goreleaser attaches the tarballs to the draft. Publishing without them
  // fails the downstream Docker build, which downloads them immediately.
  const release = getRelease(version);
  if (!release || release.assetCount === 0) {
    fail("cli-release succeeded but the draft has no assets attached");
  }
  ok(`Draft has ${release.assetCount} assets attached`);
}

async function stepPublish(version: string, autoYes: boolean): Promise<void> {
  step("Publish");
  const release = getRelease(version);
  if (!release) fail(`no release found for ${version}`);

  if (!release.isDraft) {
    ok(`${version} is already published`);
  } else {
    if (release.assetCount === 0) fail(`${version} has no assets — do not publish; the Docker build would fail`);
    if (!autoYes && !(await confirm(`Publish ${version} live now?`, false))) fail("aborted before publishing");

    const latestFlag = isPrerelease(version) ? ["--latest=false"] : ["--latest"];
    gh("release", "edit", version, "--repo", REPO, "--draft=false", ...latestFlag);
    ok(`Published ${version}`);
  }

  const url = getRelease(version)?.url ?? "";
  console.log();
  console.log(`  ${green(bold("Released:"))} ${url}`);
  console.log();
  console.log(`  Publishing fires the downstream workflows:`);
  console.log(`    ${dim("gh run list --repo " + REPO + " --workflow=cli-post-release.yml --limit 3")}`);
  console.log(`    ${dim("gh run list --repo " + REPO + " --workflow=docker-image-release.yml --limit 3")}`);
  console.log();
  console.log(`  cli-post-release promotes s3://releases.jetpack.io/devbox/stable/version,`);
  console.log(`  which is what the installer reads. Until it finishes, get.jetify.com still`);
  console.log(`  serves the previous version.`);
}

// ---------------------------------------------------------------------------
// Flows
// ---------------------------------------------------------------------------

type Mode = "draft" | "publish";

type Options = {
  mode: Mode;
  version?: string;
  title?: string;
  notesFile?: string;
  autoYes: boolean;
};

// Resuming an existing draft skips straight to the tail of the pipeline: the
// draft (and possibly the tag and build) already exist.
async function resumeDraft(version: string, opts: Options): Promise<void> {
  const release = getRelease(version);
  if (!release) fail(`no draft found for ${version}`);

  // How much of the pipeline already ran determines what's left to do, so work
  // that out before printing any step numbers.
  const needsTag = !tagPushed(version);
  const needsBuild = needsTag || release.assetCount === 0;
  totalSteps = 3 + (needsTag ? 1 : 0) + (needsBuild ? 1 : 0);

  stepPreflight({ requireCleanMain: false, checkMainCI: needsBuild });

  step("Existing draft");
  ok(`Found draft ${version} with ${release.assetCount} assets`);
  info(release.url);

  if (needsTag) {
    warn(`Tag ${version} is not pushed yet — the build has not run`);
    stepTag(version);
  } else if (needsBuild) {
    warn("Tag is pushed but no assets are attached — waiting on the build");
  } else {
    ok("Tag is pushed and artifacts are attached");
  }
  if (needsBuild) await stepWait(version);

  await stepPublish(version, opts.autoYes);
}

async function fullFlow(opts: Options): Promise<void> {
  const publishing = opts.mode === "publish";
  // preflight, version, flake, title, notes, review, draft, tag, wait [, publish]
  totalSteps = publishing ? 10 : 9;

  stepPreflight({ requireCleanMain: true, checkMainCI: true });
  const version = await stepChooseVersion(opts.version);
  await stepFlakeBump(version, opts.autoYes);
  const title = await stepTitle(version, opts.title);
  const notes = await stepNotes(version, opts.notesFile);
  await stepConfirm(version, title, notes, opts.mode, opts.autoYes);
  stepDraft(version, title, notes);
  stepTag(version);
  await stepWait(version);

  if (publishing) {
    await stepPublish(version, opts.autoYes);
    return;
  }

  step("Done");
  ok(`Draft ${version} is built and ready for review.`);
  console.log();
  console.log(`  ${getRelease(version)?.url ?? ""}`);
  console.log();
  console.log(`  When you're happy with it, take it live with:`);
  console.log(`    ${bold("devbox run publish-release")}`);
}

async function publishFlow(opts: Options): Promise<void> {
  // Publishing can either finish an existing draft or run the whole pipeline.
  // Offer the choice up front rather than guessing.
  if (!opts.version) {
    const drafts = listDrafts();
    if (drafts.length > 0) {
      console.log(`\n${bold("Unpublished drafts")}`);
      for (const d of drafts) console.log(`  • ${d.tagName} ${dim(`(created ${d.createdAt.slice(0, 10)})`)}`);
      console.log();

      const options = [
        ...drafts.map((d) => ({ value: d.tagName, label: `Publish the existing draft ${bold(d.tagName)}` })),
        { value: "__new__", label: "Start a new release from scratch" },
      ];
      const picked = await choose("What do you want to publish?", options);
      if (picked !== "__new__") {
        await resumeDraft(picked, opts);
        return;
      }
    }
  } else if (getRelease(opts.version)?.isDraft) {
    await resumeDraft(validateVersion(opts.version), opts);
    return;
  }

  await fullFlow(opts);
}

// ---------------------------------------------------------------------------
// CLI
// ---------------------------------------------------------------------------

function usage(): void {
  console.log(`
${bold("Devbox release")}

  ${bold("devbox run draft-release")}     Build a release and leave it as a draft
  ${bold("devbox run publish-release")}   Build a release and take it live, or
                                 publish a draft that already exists

Both walk every step in order and prompt for the tag, title and description.

${bold("Direct invocation")}

  node scripts/release.ts --draft   [flags]
  node scripts/release.ts --publish [flags]
  node scripts/release.ts --changes            What would ship, and the
                                               recommended version bump
  node scripts/release.ts --status  [version]  Where a release currently stands

${bold("Flags")}   (each one skips the prompt it answers)

  --version <x.y.z>    Version to release, bare semver, no leading "v"
  --title <string>     Release title
  --notes-file <path>  Release notes, instead of opening \$EDITOR
  --yes                Skip confirmations (for scripted runs)
`);
}

// Read-only view of what a release would contain. Nothing here touches the
// remote, so it's safe to run at any time.
function changesReport(): void {
  const prev = previousTag();
  const commits = commitsSince(prev);
  if (commits.length === 0) {
    console.log(`No commits since ${prev} — nothing to release.`);
    return;
  }
  summarizeChanges(prev, commits);

  const rec = recommendVersion(prev, commits);
  console.log();
  console.log(`  ${bold("Recommended")}  ${green(rec.version)} (${rec.bump} bump — ${rec.reason})`);
  console.log(`  ${bold("flake.nix")}    lastTag = ${flakeLastTag()}${flakeLastTag() === rec.version ? "" : dim(" (needs bumping)")}`);
}

function statusReport(versionArg?: string): void {
  const version = versionArg ? validateVersion(versionArg) : previousTag();
  const release = getRelease(version);
  const runs = ghJSON<{ status: string; conclusion: string | null }[]>(
    "run", "list", "--repo", REPO, "--workflow=cli-release.yml",
    "--branch", version, "--event", "push", "--limit", "1", "--json", "status,conclusion",
  );

  console.log(`Version:        ${version}`);
  console.log(`Previous tag:   ${previousTag(version)}`);
  console.log(`flake lastTag:  ${flakeLastTag()}`);
  console.log(`Tag pushed:     ${tagPushed(version) ? "yes" : "no"}`);
  console.log(
    `Release:        ${
      release
        ? `exists (draft=${release.isDraft}, prerelease=${release.isPrerelease}, assets=${release.assetCount})`
        : "none"
    }`,
  );
  console.log(`cli-release:    ${runs[0] ? `${runs[0].status}/${runs[0].conclusion ?? "-"}` : "none"}`);
}

async function main(): Promise<void> {
  const argv = process.argv.slice(2);
  if (argv.length === 0 || argv.includes("--help") || argv.includes("-h")) {
    usage();
    return;
  }

  const flag = (name: string): string | undefined => {
    const i = argv.indexOf(`--${name}`);
    if (i === -1) return undefined;
    const value = argv[i + 1];
    if (!value || value.startsWith("--")) fail(`--${name} requires a value`);
    return value;
  };

  if (argv[0] === "--status") {
    statusReport(argv[1]);
    return;
  }

  if (argv[0] === "--changes") {
    changesReport();
    return;
  }

  const draft = argv.includes("--draft");
  const publish = argv.includes("--publish");
  if (draft && publish) fail("pass either --draft or --publish, not both");
  if (!draft && !publish) {
    usage();
    fail(`unknown arguments: ${argv.join(" ")}`);
  }

  const opts: Options = {
    mode: publish ? "publish" : "draft",
    version: flag("version"),
    title: flag("title"),
    notesFile: flag("notes-file"),
    autoYes: argv.includes("--yes"),
  };

  if (opts.version) validateVersion(opts.version);
  if (opts.notesFile && !fs.existsSync(opts.notesFile)) fail(`no such file: ${opts.notesFile}`);

  console.log(bold(`\nDevbox release — ${opts.mode === "publish" ? "publish" : "draft"} mode`));

  if (opts.mode === "publish") await publishFlow(opts);
  else await fullFlow(opts);
}

main()
  .then(() => {
    closeReader();
    process.exit(0);
  })
  .catch((err: unknown) => {
    closeReader();
    console.error(`\n${red("error:")} ${err instanceof Error ? err.message : String(err)}\n`);
    process.exit(1);
  });
