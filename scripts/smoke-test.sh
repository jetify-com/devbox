#!/usr/bin/env bash
#
# Smoke-tests a devbox release build the way users get it: installs it with
# install.sh, then checks the version it reports and, optionally, that it can
# create a project, add a package and run it.
#
# Usage: scripts/smoke-test.sh [--project] <dist-dir> <expected-version>
#
# <dist-dir> must contain checksums.txt and devbox_<version>_<os>_<arch>.tar.gz
# for this machine, as goreleaser writes them. install.sh normally downloads
# both from GitHub releases; here a curl shim on PATH serves them from
# <dist-dir> instead, so the real install.sh runs unmodified (including its
# version detection and checksum verification) without publishing anything.
#
# --project also runs `devbox init`, `devbox add hello` and `devbox run hello`,
# which needs Nix.
#
# Installs to /usr/local/bin/devbox (with sudo if needed), so only run it on a
# throwaway machine such as a CI runner.

set -euo pipefail

run_project=0
if [[ ${1-} == "--project" ]]; then
	run_project=1
	shift
fi
if [[ $# -ne 2 ]]; then
	echo "usage: $0 [--project] <dist-dir> <expected-version>" >&2
	exit 2
fi

dist_dir="$(cd "$1" && pwd)"
expected="$2"
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
releases_url="https://github.com/jetify-com/devbox/releases"

fail() {
	echo "::error::smoke test: $*" >&2
	exit 1
}

group() { echo "::group::$*"; }
endgroup() { echo "::endgroup::"; }

[[ -f "${dist_dir}/checksums.txt" ]] || fail "${dist_dir}/checksums.txt is missing"

work="$(mktemp -d)"
trap 'rm -rf "${work}"' EXIT

# ---------------------------------------------------------------------------
# install.sh
# ---------------------------------------------------------------------------
group "Install with install.sh"
mkdir -p "${work}/shim"
cat >"${work}/shim/curl" <<EOF
#!/usr/bin/env bash
# Serves devbox release downloads from ${dist_dir}; see scripts/smoke-test.sh.
set -euo pipefail
out=""
url=""
while [[ \$# -gt 0 ]]; do
	case "\$1" in
	--output | -o) out="\$2"; shift 2 ;;
	-*) shift ;;
	*) url="\$1"; shift ;;
	esac
done
case "\${url}" in
"${releases_url}/latest/download/checksums.txt")
	src="${dist_dir}/checksums.txt" ;;
"${releases_url}/download/"*/*)
	src="${dist_dir}/\${url##*/}" ;;
*)
	echo "curl shim: unexpected URL \${url}" >&2
	exit 22 ;;
esac
echo "curl shim: \${url} -> \${src}" >&2
[[ -f "\${src}" ]] || { echo "curl shim: \${src} not found" >&2; exit 22; }
cp "\${src}" "\${out}"
EOF
chmod +x "${work}/shim/curl"

# Start from a clean state so the launcher can't pick up a version from an
# earlier install on this machine.
rm -f "${XDG_CACHE_HOME:-${HOME}/.cache}/devbox/current-version" \
	"${XDG_CACHE_HOME:-${HOME}/.cache}/devbox/latest-version"

install_log="${work}/install.log"
if ! PATH="${work}/shim:${PATH}" FORCE=1 bash "${repo_root}/install.sh" 2>&1 | tee "${install_log}"; then
	fail "install.sh failed"
fi
# Strip install.sh's colors before matching its output.
sed $'s/\x1b\\[[0-9;]*[A-Za-z]//g' "${install_log}" | grep -qF "Successfully installed devbox ${expected}" ||
	fail "install.sh didn't report installing devbox ${expected}"
endgroup

devbox_bin="/usr/local/bin/devbox"
[[ -x ${devbox_bin} ]] || fail "install.sh didn't install ${devbox_bin}"

# ---------------------------------------------------------------------------
# Version
# ---------------------------------------------------------------------------
group "Check version"
# Run it like a user would: no DEVBOX_USE_VERSION, so the launcher picks the
# version that install.sh recorded. If that handoff breaks, the launcher would
# go to GitHub for the latest release and report that version instead.
got="$("${devbox_bin}" version)"
echo "devbox version: ${got}"
[[ ${got} == "${expected}" ]] || fail "devbox version reported '${got}', want '${expected}'"

verbose="$("${devbox_bin}" version --verbose)"
echo "${verbose}"
grep -Eq "^Version: +${expected//./\\.}$" <<<"${verbose}" || fail "devbox version --verbose doesn't report ${expected}"
grep -Eq "^Commit: +[0-9a-f]{7,}$" <<<"${verbose}" || fail "devbox version --verbose doesn't report a commit"
grep -Eq "^Launcher: +[0-9]+\.[0-9]+\.[0-9]+$" <<<"${verbose}" ||
	fail "devbox version --verbose doesn't report a launcher version; is the build missing -X ...build.Launcher=true?"
endgroup

if [[ ${run_project} -ne 1 ]]; then
	echo "Smoke test passed (install and version only)."
	exit 0
fi

# ---------------------------------------------------------------------------
# Project
# ---------------------------------------------------------------------------
group "devbox init / add / run / rm"
project="${work}/project"
mkdir -p "${project}"
cd "${project}"

"${devbox_bin}" init
[[ -f devbox.json ]] || fail "devbox init didn't create devbox.json"

"${devbox_bin}" add hello
grep -q '"hello' devbox.json || fail "devbox add hello didn't add hello to devbox.json"
[[ -f devbox.lock ]] || fail "devbox add hello didn't create devbox.lock"
grep -q '"hello@latest"' devbox.lock || fail "devbox.lock has no hello@latest entry"

out="$("${devbox_bin}" run hello)"
echo "${out}"
[[ ${out} == *"Hello, world!"* ]] || fail "devbox run hello printed '${out}'"

# devbox run sets up the project's environment for arbitrary commands too.
# shellcheck disable=SC2016 # expanded by the inner shell
"${devbox_bin}" run -- sh -c 'test -f "${DEVBOX_PROJECT_ROOT:-/nonexistent}/devbox.json"' ||
	fail "DEVBOX_PROJECT_ROOT doesn't point at the project inside devbox run"

"${devbox_bin}" rm hello
if grep -q '"hello' devbox.json; then
	fail "devbox rm hello left hello in devbox.json"
fi
endgroup

echo "Smoke test passed."
