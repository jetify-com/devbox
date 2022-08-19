# Devbox: Instant development environments and containers using `nix`.

Devbox is a tool that lets you easily manage development environments.

## Prerequisites
* [Nix Package Manager](https://nixos.org/download.html) (Don't worry, you don't need to learn Nix)
* [Docker](https://docs.docker.com/engine/install/)

## Quickstart

Initialize Devbox for your project
```bash
devbox init
```

Add [Nix Packages](https://search.nixos.org/packages) to your project
```bash
devbox add hello go-rice
```

Start a local development shell with your project and packages
```bash
devbox shell
```

Build a Docker image of your project
```bash
devbox build
```
## Language Support

Devbox can detect and automatically configure shells + Docker containers for your language. 

To view the current plan for your project run: 
```bash
devbox plan
```
Currently supported languages include: 
* Go

## Related Work

- [nix](https://nixos.org/)
