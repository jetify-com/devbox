---
title: Quickstart
sidebar_position: 1
hide_title: false
---

Jetify Devspace let you spin up reproducible cloud development environments in your browser in seconds. Jetify Devspace is powered by [Devbox](/devbox/docs), so you can run your environment on any machine. You can launch any

Let's launch our first Dev Environment in Jetify Devspace.

## Step 1: Launch Jetify Devspace from Github

You can launch any Github Repo in a Jetify Sandbox by prepend the repo URL with:

```bash
https://cloud.jetify.com/new/
```

For example, to launch the [Devbox repo](https://github.com/jetify-com/devbox), open the following URL in your browser

  [https://cloud.jetify.com/new/github.com/jetify-com/devbox](https://cloud.jetify.com/new/github.com/jetify-com/devbox)

:::tip
  If you need some inspiration, you can also launch one of our [templates](/devbox/docs/devbox_examples) projects to get started
:::

You can also launch a new Devspace by navigating to your [Dashboard](https://cloud.jetify.com/dashboard) and clicking on the `New Devspace` button.

## Step 2: Customize your Environment with Devbox

You can customize your Jetify Devspace with over 100,000 Nix packages in seconds using Devbox.

If your project doesn't already have a devbox.json, you can initialize one with:

```bash
devbox init
```

Once initialized, you can install your packages using:

```bash
devbox add <package>@<version>
```

For example, to install `python 3.11`, you can run:

```bash
devbox add python@3.11
```

You can find packages to install using `devbox search <package>`, or by searching in your browser with [Nixhub](https://www.nixhub.io)

Packages you install will be added to your `devbox.json` file. You can also use this `devbox.json` file configure your environment with [scripts](/devbox/docs/guides/scripts), [services](/devbox/docs/guides/services), and more

For further reading on how to install packages with Devbox, see:

* [Devbox Quickstart](/devbox/docs/quickstart)
* [Devbox CLI Reference](/devbox/docs/cli_reference/devbox)

## Step 3: Save your Dev Environment with `devbox.json`

Once you've customized your environment, you can save your Dev Environment config to source control by checking in your `devbox.json` and `devbox.lock` files. These files can be used to recreate your environment on Jetify Devspace, or on any other machine that has devbox installed.

You can also use this file to configure initialization hooks, scripts, services, and environment variables for your project. For further reading, see:

* [Devbox Configuration Reference](/devbox/docs/configuration)
* [Devbox Script](/devbox/docs/guides/scripts)
* [Devbox Services](/devbox/docs/guides/services)
