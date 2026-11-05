#!/bin/bash
#
# Install script
#
# Downloads the latest devbox release and installs its binary. Release builds
# of devbox are their own launcher: the installed binary picks the devbox
# version to run, downloads it if needed, and checks for updates.

set -euo pipefail

# ========================
# Customize install script
# ========================

# This script is published at get.jetify.com/devbox so users can install via:
# curl -fsSL https://get.jetify.com/devbox | bash

readonly INSTALL_DIR="/usr/local/bin"
readonly BIN="devbox"
readonly RELEASES_URL="https://releases.jetify.com/devbox"
readonly DOWNLOAD_URL="https://github.com/jetify-com/devbox/releases/download"
# Must match the version files that devbox reads (internal/launcher/files.go).
readonly CACHE_DIR="${XDG_CACHE_HOME:-${HOME}/.cache}/${BIN}"

readonly TITLE="Devbox 📦 by Jetify"
readonly DESCRIPTION=$(
	cat <<EOF
  Instant, easy and predictable development environments.

  This script downloads and installs the latest devbox binary.
EOF
)
readonly DOCS_URL="https://github.com/jetify-com/devbox"
readonly COMMUNITY_URL="https://discord.gg/jetify"

# ====================
# flags.sh
# ====================
FORCE="${FORCE:-0}"

parse_flags() {
	while [ "$#" -gt 0 ]; do
		case "$1" in
		-f | --force)
			FORCE=1
			shift 1
			;;
		*)
			error "Unknown option: $1"
			exit 1
			;;
		esac
	done
}

# ====================
# format.sh
# ====================

readonly BOLD="$(tput bold 2>/dev/null || echo '')"
readonly GREY="$(tput setaf 8 2>/dev/null || echo '')"
readonly UNDERLINE="$(tput smul 2>/dev/null || echo '')"
readonly RED="$(tput setaf 1 2>/dev/null || echo '')"
readonly GREEN="$(tput setaf 2 2>/dev/null || echo '')"
readonly YELLOW="$(tput setaf 3 2>/dev/null || echo '')"
readonly BLUE="$(tput setaf 4 2>/dev/null || echo '')"
readonly MAGENTA="$(tput setaf 5 2>/dev/null || echo '')"
readonly CYAN="$(tput setaf 6 2>/dev/null || echo '')"
readonly NO_COLOR="$(tput sgr0 2>/dev/null || echo '')"
readonly CLEAR_LAST_MSG="\033[1F\033[0K"

title() {
	local -r text="$*"
	printf "%s\n" "${BOLD}${MAGENTA}${text}${NO_COLOR}"
}

header() {
	local -r text="$*"
	printf "%s\n" "${BOLD}${text}${NO_COLOR}"
}

plain() {
	local -r text="$*"
	printf "%s\n" "${text}"
}

warn() {
	printf "%s\n" "${YELLOW}! $*${NO_COLOR}"
}

error() {
	local -r text="$*"
	printf "%s\n" "${RED}✘ ${text}${NO_COLOR}" >&2
}

success() {
	local -r text="$*"
	printf "%s\n" "${GREEN}✓${NO_COLOR} ${text}"
}

start_task() {
	local -r text="$*"
	printf "%s\n" "${BOLD}${GREY}→${NO_COLOR} ${text}..."
}

end_task() {
	local -r text="$*"
	printf "${CLEAR_LAST_MSG}%s\n" "${GREEN}✓${NO_COLOR} ${text}... [DONE]"
}

fail_task() {
	local -r text="$*"
	printf "${CLEAR_LAST_MSG}%s\n" "${RED}✘ ${text}... [FAILED]${NO_COLOR}" >&2
}

confirm() {
	if [ ${FORCE-} -ne 1 ]; then
		printf "%s " "${MAGENTA}?${NO_COLOR} $* ${BOLD}[Y/n]${NO_COLOR}"
		set +e
		read -r yn </dev/tty
		rc=$?
		set -e
		if [ $rc -ne 0 ]; then
			error "Error reading from prompt (re-run with '-f' flag to auto select Yes if running in a script)"
			exit 1
		fi
		if [ "$yn" != "y" ] && [ "$yn" != "Y" ] && [ "$yn" != "yes" ] && [ "$yn" != "" ]; then
			error 'Aborting (please answer "yes" to continue)'
			exit 1
		fi
	fi
}

# =========
# util.sh
# =========
has() {
	command -v "$1" 1>/dev/null 2>&1
}

download() {
	local -r url="$1"
	local -r file="$2"
	local cmd=""

	if has curl; then
		cmd="curl --fail --silent --location --output $file $url"
	elif has wget; then
		cmd="wget --quiet --output-document=$file $url"
	elif has fetch; then
		cmd="fetch --quiet --output=$file $url"
	else
		error "No program to download files found. Please install one of: curl, wget, fetch"
		exit 1
	fi

	$cmd && return 0 || rc=$?
	error "Command failed (exit code $rc): ${BLUE}${cmd}${NO_COLOR}"
	exit $rc
}

sha256() {
	if has sha256sum; then
		sha256sum "$1" | cut -d ' ' -f 1
	elif has shasum; then
		shasum -a 256 "$1" | cut -d ' ' -f 1
	else
		error "No program to verify checksums found. Please install one of: sha256sum, shasum"
		exit 1
	fi
}

# platform prints the platform part of release archive names, such as
# darwin_arm64 or linux_amd64.
platform() {
	local os arch
	os="$(uname -s | tr '[:upper:]' '[:lower:]')"
	arch="$(uname -m | tr '[:upper:]' '[:lower:]')"
	case "${arch}" in
	x86_64 | amd64) arch="amd64" ;;
	aarch64 | arm64) arch="arm64" ;;
	armv7*) arch="armv7l" ;;
	i386 | i686) arch="386" ;;
	esac
	case "${os}" in
	linux | darwin) ;;
	*)
		error "Unsupported operating system: ${os}"
		exit 1
		;;
	esac
	printf '%s_%s' "${os}" "${arch}"
}

# ==============
# Implementation
# ==============
intro_msg() {
	title "${TITLE}"
	plain "${DESCRIPTION}"
	printf "\n"
	header "Confirm Installation Details"
	plain "  Location:     ${GREEN}${INSTALL_DIR}/${BIN}${NO_COLOR}"
	plain "  Download URL: ${UNDERLINE}${BLUE}${DOWNLOAD_URL}${NO_COLOR}"
	printf "\n"
}

install_flow() {
	confirm "Install ${GREEN}${BIN}${NO_COLOR} to ${GREEN}${INSTALL_DIR}${NO_COLOR} (requires sudo)?"
	printf "\n"
	header "Downloading and Installing"

	TMP_DIR="$(mktemp -d)"
	trap 'rm -rf "${TMP_DIR}"' EXIT
	local -r tmp_dir="${TMP_DIR}"

	start_task "Finding the latest version"
	download "${RELEASES_URL}/stable/version" "${tmp_dir}/version"
	local version
	version="$(tr -d '[:space:]' <"${tmp_dir}/version")"
	if [[ ! ${version} =~ ^v?[0-9]+\.[0-9]+\.[0-9]+ ]]; then
		fail_task "Finding the latest version"
		error "Unexpected version: ${version}"
		exit 1
	fi
	version="${version#v}"
	end_task "Finding the latest version (${version})"

	local -r archive="${BIN}_${version}_$(platform).tar.gz"
	start_task "Downloading ${BIN} ${version}"
	download "${DOWNLOAD_URL}/${version}/checksums.txt" "${tmp_dir}/checksums.txt"
	download "${DOWNLOAD_URL}/${version}/${archive}" "${tmp_dir}/${archive}"
	end_task "Downloading ${BIN} ${version}"

	start_task "Verifying checksum"
	local -r expected="$(awk -v name="${archive}" '$2 == name { print $1 }' "${tmp_dir}/checksums.txt")"
	local -r actual="$(sha256 "${tmp_dir}/${archive}")"
	if [[ -z ${expected} || ${actual} != "${expected}" ]]; then
		fail_task "Verifying checksum"
		plain "  Expected checksum (sha256): ${expected:-none}"
		plain "  Actual checksum (sha256):   ${actual}"
		exit 1
	fi
	end_task "Verifying checksum"

	mkdir "${tmp_dir}/unpacked"
	tar -xzof "${tmp_dir}/${archive}" -C "${tmp_dir}/unpacked"
	chmod 755 "${tmp_dir}/unpacked/${BIN}"

	start_task "Installing in ${INSTALL_DIR}/${BIN} (requires sudo)"
	# Only use sudo when the install dir (or its closest existing parent, if
	# it doesn't exist yet) isn't writable.
	local writable_dir="${INSTALL_DIR}"
	while [[ ! -e ${writable_dir} ]]; do
		writable_dir="$(dirname "${writable_dir}")"
	done
	local sudo=""
	if [[ ! -w ${writable_dir} ]]; then
		sudo="$(command -v sudo || true)"
	fi
	# Copy next to the destination and rename, so that a devbox command that
	# starts during the install never runs a partial binary.
	${sudo} bash -c 'mkdir -p "$1" && cp "$2" "$1/.$3.tmp" && mv -f "$1/.$3.tmp" "$1/$3"' \
		bash "${INSTALL_DIR}" "${tmp_dir}/unpacked/${BIN}" "${BIN}"
	end_task "Installing in ${INSTALL_DIR}/${BIN}"

	# Run the installed version from now on, and don't wait to check for a
	# newer one on the first run.
	if mkdir -p "${CACHE_DIR}" 2>/dev/null &&
		printf '%s\n' "${version}" >"${CACHE_DIR}/current-version" &&
		printf '%s\n' "${version}" >"${CACHE_DIR}/latest-version"; then
		:
	else
		warn "Couldn't write ${CACHE_DIR}/current-version. ${BIN} will check for the latest version on its first run."
	fi

	success "${BOLD}Successfully installed ${GREEN}${BIN} ${version}${NO_COLOR}${BOLD}${NO_COLOR} 🚀"
	printf "\n"
}

next_steps_msg() {
	header "Next Steps"
	plain "  1. ${BOLD}Learn how to use ${BIN}${NO_COLOR}"
	plain "     ${GREY}Run ${CYAN}${BIN} help${GREY} or read the docs at ${UNDERLINE}${BLUE}${DOCS_URL}${NO_COLOR}"
	plain "  2. ${BOLD}Get help and give feedback${NO_COLOR}"
	plain "     ${GREY}Join our community at ${UNDERLINE}${BLUE}${COMMUNITY_URL}${NO_COLOR}"
}

main() {
	parse_flags "$@"
	intro_msg
	install_flow
	next_steps_msg
}

main "$@"
