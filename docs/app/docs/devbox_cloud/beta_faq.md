---
title: Devbox Playground FAQ
sidebar_position: 4
---

### What do I need to use a Devbox playground?

To start a Devbox playground from your Browser, you will need a Github Account.

### Does my project need to use Devbox to use Devbox playground?

While you can open any Github Repo in a Devbox playground, you will need a `devbox.json` to install packages or configure the environment. You can add any packages in your shell by running `devbox add <pkg>`

### Can I use my own IDE or editor with a Devbox playground?

Devbox.sh provides a Cloud IDE that you can use to edit your projects in the browser, but you can also open your project in your local VSCode Editor by clicking the `Open in Desktop` button.

You can also use your own tools when you connect to the VM via SSH. See our [Getting Started Guide](index.mdx) for more details.

### Do I have to pay to use Devbox.sh?

Devbox.sh is free to use during the Beta period, subject to the restrictions described below.

### What are the resource limits for Devbox playgrounds

* **CPU**: 4 Cores
* **RAM**: 8 GB
* **SSD**: 8 GB

If you are interested in using Devbox playgrounds or CDE in an enterprise setting, please reach out to us at [info@jetify.com](mailto://info@jetify.com)

### Is there a time limit on Devbox playgrounds?

Your playground will be suspended after 4 hours of inactivity, and can be restarted by reopening the playground from your [dashboard](https://devbox.sh/app/projects).

playgrounds are also deleted every 12 hours, regardless of activity

### I want to request more resources, persistence, or a different OS for my VM

Please contact us at info@jetify.com if you are interested in a custom solution for your enterprise.

### What OS does the Devbox.sh use?

Debian Linux, running on a x86-64 platform

### How many VM's can I run concurrently?

You can have up to 5 concurrent projects per Github Account. To run more playgrounds, you can visit your [Dashboard](https://devbox.sh/app/projects) to delete older playgrounds

### Where does Devbox run my playground?

Devbox VMs are run as Fly Machines in local Data Centers. To minimize latency, Devbox.sh will attempt to create a Fly Machine as close to your current location as possible.


