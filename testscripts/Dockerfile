# syntax=docker/dockerfile:1

# This Dockerfile is designed to be as distro-agnostic as possible. By default
# it uses ubuntu:noble as its base image, but this can be overridden by setting
# the BASEIMAGE and BASETAG build args. This makes it easier to run Devbox tests
# against different distros/versions without creating a bunch of separate
# Dockerfiles.

ARG BASEIMAGE=ubuntu
ARG BASETAG=noble

FROM scratch AS installer

ADD --chmod=0755 --link https://install.determinate.systems/nix/nix-installer-x86_64-linux /nix-installer-linux-amd64
ADD --chmod=0755 --link https://install.determinate.systems/nix/nix-installer-aarch64-linux /nix-installer-linux-arm64

FROM $BASEIMAGE:$BASETAG

ARG TARGETOS
ARG TARGETARCH

ENV XDG_CACHE_HOME=/nix/cache

COPY --from=installer --link /nix-installer-$TARGETOS-$TARGETARCH nix-installer
RUN <<EOF
set -e

mkdir -p "$XDG_CACHE_HOME"

# Setting this env var is the same as passing --extra-conf to nix-installer.
export NIX_INSTALLER_EXTRA_CONF="filter-syscalls = false
sandbox = false
experimental-features = nix-command flakes fetch-closure ca-derivations
use-xdg-base-directories = true"

./nix-installer install linux --no-confirm --logger full --init none
rm nix-installer
EOF
ENV SSL_CERT_FILE=/nix/var/nix/profiles/default/etc/ssl/certs/ca-bundle.crt

COPY --link */*.test.txt /devbox/testscripts/
COPY --link testscripts-$TARGETOS-$TARGETARCH /devbox/testscripts/test

VOLUME "/nix"
WORKDIR /devbox/testscripts
ENTRYPOINT [ "/devbox/testscripts/test" ]
