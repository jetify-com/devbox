---
title: Using the Jetify Prebuilt Cache
---

The Jetify Prebuilt Cache provides users with prebuilt binaries of popular packages for the most common OS(Linux, macOS) + Architecture (x86-64, aarch64) combinations. 

The Jetify Prebuilt cache is intended to supplement the official NixOS Cache, and includes packages which are not available by default. This includes: 

1. Older packages that have been garbage collected
2. Packages which have not been built for certain platforms
3. Packages with unfree licenses, which are not automatically built by NixOS

## Using the Prebuilt Cache

The Prebuilt Cache is available for free to every developer who signs up for a Jetify Cloud account. Devbox will automatically configure itself to use the Prebuilt Cache when you login with `devbox auth login`: no additional action or steps are required. 

:::info 
Free Jetify Cloud accounts are restricted to a **25 GB per month** download limit, and cannot generate access tokens for the cache. 

Solo, Starter, and Scaleup accounts have unlimited access to the Prebuilt Cache.
::: 

## Packages included in the Prebuilt Cache

Some of the packages included in the Prebuilt Cache are:

* MongoDB
* Terraform
* Vault
* DynamoDB local
* Pulumi
* Helm
* Unrar
* Graphite 

More packages are added regularly. If you encounter a package that you think should be in the Prebuilt Cache, notify us on our [Discord](https://discord.gg/jetify). 