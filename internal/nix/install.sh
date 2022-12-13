#!/bin/sh

# This script installs the Nix package manager on your system by
# downloading a binary distribution and running its installer script
# (which in turn creates and populates /nix).

{ # Prevent execution if this script was only partially downloaded
oops() {
    echo "$0:" "$@" >&2
    exit 1
}

umask 0022

tmpDir="$(mktemp -d -t nix-binary-tarball-unpack.XXXXXXXXXX || \
          oops "Can't create temporary directory for downloading the Nix binary tarball")"
cleanup() {
    rm -rf "$tmpDir"
}
trap cleanup EXIT INT QUIT TERM

require_util() {
    command -v "$1" > /dev/null 2>&1 ||
        oops "you do not have '$1' installed, which I need to $2"
}

case "$(uname -s).$(uname -m)" in
    Linux.x86_64)
        hash=c6d48479d50a01cdfc3669440776692ca7094ff29028b1fec6da0abeead16a01
        path=sicsy40akh9hs5r8iz1rkgnh46yfns4h/nix-2.11.1-x86_64-linux.tar.xz
        system=x86_64-linux
        ;;
    Linux.i?86)
        hash=37fa1567394baf7fac2651d4d60890191b9d183626faf2069c59dbc602136ac5
        path=17s133r9xnvwkg7v6var7i24w2av285g/nix-2.11.1-i686-linux.tar.xz
        system=i686-linux
        ;;
    Linux.aarch64)
        hash=b8ef85ea43d30ed89fdd176d8b044c5d1301628a77886742ea3f684cd4dc6db3
        path=fhi79b7zvz6np2rhld034qzpf7pfmb67/nix-2.11.1-aarch64-linux.tar.xz
        system=aarch64-linux
        ;;
    Linux.armv6l_linux)
        hash=7181889751cb83add9b4ef7ea6bd0adb90eb0cd8c78422315cc22e6a7188dafd
        path=r44jpi4hki9llkznyxdl18a7f634an2p/nix-2.11.1-armv6l-linux.tar.xz
        system=armv6l-linux
        ;;
    Linux.armv7l_linux)
        hash=8550980d6001f42f5dd2e969a016304f3e659bdcd9146e04c18b02f5b63994cd
        path=0kmd8q1g4gyw6pr474xp3nxs8mvyigqh/nix-2.11.1-armv7l-linux.tar.xz
        system=armv7l-linux
        ;;
    Darwin.x86_64)
        hash=16dac47e397ff9026af23f355cc84465a2af7ec56b65ddb52c8b124d700556b1
        path=f0zhwzkvn5vv583mzbj0dqahzcajkglx/nix-2.11.1-x86_64-darwin.tar.xz
        system=x86_64-darwin
        ;;
    Darwin.arm64|Darwin.aarch64)
        hash=ddead2fa8ef6b9b58fec4ab12b460f802a962fff68fb1c8fa47c7b8b5739bc0b
        path=d2gq40kvzckjmhwbbnh718w64v6zlr3m/nix-2.11.1-aarch64-darwin.tar.xz
        system=aarch64-darwin
        ;;
    *) oops "sorry, there is no binary distribution of Nix for your platform";;
esac

# Use this command-line option to fetch the tarballs using nar-serve or Cachix
if [ "${1:-}" = "--tarball-url-prefix" ]; then
    if [ -z "${2:-}" ]; then
        oops "missing argument for --tarball-url-prefix"
    fi
    url=${2}/${path}
    shift 2
else
    url=https://releases.nixos.org/nix/nix-2.11.1/nix-2.11.1-$system.tar.xz
fi

tarball=$tmpDir/nix-2.11.1-$system.tar.xz

require_util tar "unpack the binary tarball"
if [ "$(uname -s)" != "Darwin" ]; then
    require_util xz "unpack the binary tarball"
fi

if command -v curl > /dev/null 2>&1; then
    fetch() { curl --fail -L "$1" -o "$2"; }
elif command -v wget > /dev/null 2>&1; then
    fetch() { wget "$1" -O "$2"; }
else
    oops "you don't have wget or curl installed, which I need to download the binary tarball"
fi

echo "downloading Nix 2.11.1 binary tarball for $system from '$url' to '$tmpDir'..."
fetch "$url" "$tarball" || oops "failed to download '$url'"

if command -v sha256sum > /dev/null 2>&1; then
    hash2="$(sha256sum -b "$tarball" | cut -c1-64)"
elif command -v shasum > /dev/null 2>&1; then
    hash2="$(shasum -a 256 -b "$tarball" | cut -c1-64)"
elif command -v openssl > /dev/null 2>&1; then
    hash2="$(openssl dgst -r -sha256 "$tarball" | cut -c1-64)"
else
    oops "cannot verify the SHA-256 hash of '$url'; you need one of 'shasum', 'sha256sum', or 'openssl'"
fi

if [ "$hash" != "$hash2" ]; then
    oops "SHA-256 hash mismatch in '$url'; expected $hash, got $hash2"
fi

unpack=$tmpDir/unpack
mkdir -p "$unpack"
tar -xJf "$tarball" -C "$unpack" || oops "failed to unpack '$url'"

script=$(echo "$unpack"/*/install)

[ -e "$script" ] || oops "installation script is missing from the binary tarball!"
export INVOKED_FROM_INSTALL_IN=1
"$script" "$@"

} # End of wrapping
