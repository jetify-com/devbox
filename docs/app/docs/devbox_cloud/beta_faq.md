---
title: Devbox Cloud Open Beta FAQ
sidebar_position: 4
---

### What do I need to use Devbox Cloud?

To use Devbox Cloud from your Browser, you will need a Github Account.

To use Devbox Cloud from your CLI, you will need:

* A Github account with linked SSH keys
* Devbox 0.3.0 or later

### Does my project need to use Devbox to use Devbox Cloud?

While you can open any Github Repo in a Devbox Cloud Shell, you will need a `devbox.json` to install packages or configure the environment. You can create a `devbox.json` in your shell by running `devbox init`

### Can I use my own IDE or editor with Devbox Cloud?

Devbox.sh provides a Cloud IDE that you can use to edit your projects in the browser, but you can also open your project in your local VSCode Editor by clicking the open in desktop button.

You can also use your own tools when you start Devbox Cloud from the terminal or SSH. See our [Getting Started Guide](getting_started.mdx) for more details.

### Do I have to pay to use Devbox Cloud during the Open Beta?

Devbox Cloud is free to use during the Open Beta period, subject to the restrictions listed below. We expect to continue offering a free tier for personal use after the Open Beta period, but we will offer Paid Plans that provide more resources, projects, and persistence.

### What are the resource limits for Devbox Cloud VMs

* **CPU**: 4 Cores
* **RAM**: 8 GB
* **SSD**: 8 GB

If you need additional resources for your project, please reach out to us for **[Early Access](https://jetpack-io.typeform.com/devbox-cloud)**

### I want to request more resources, persistence, or a different OS for my VM

Future releases will add more flexibility and features as part of our paid plans. If you'd like to sign up for early access to these plans, please sign up for **[Early Access](https://jetpack-io.typeform.com/devbox-cloud)**

### What OS does Devbox Cloud use?

Debian Linux, running on a x86-64 platform

### How many VM's can I run concurrently?

You can have up to 5 concurrent projects per Github Account.

### How long will my Devbox Cloud Shell stay alive for?

VMs will stay alive for up to 8 hours after going idle. After that point, the VM and all data will be deleted.

### Where will Devbox run my VM?

Devbox VMs are run as Fly Machines in local Data Centers. To minimize latency, Devbox Cloud will attempt to create a Fly Machine as close to your current location as possible.


