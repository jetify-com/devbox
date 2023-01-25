---
title: Devbox Cloud Beta
---

Devbox Cloud is a new way to create and run your Devbox Project in an isolated cloud environment. Devbox Cloud let's you quickly spin up an on-demand Linux Edge VM with your Devbox dependencies and shell, using either a local project or your browser.

:::info
Devbox Shell is currently available in Open Beta, and is under active development. Please see the information in [What You Need To Know](#what-you-need-to-know) below for more details
:::

Devbox Cloud's Open Beta is available to any developer using Devbox 0.2.3 or higher. 
* To get started with Devbox Cloud from your terminal, visit our [Quickstart](getting_started.md). 
* To learn how to use Devbox Cloud from your browser, visit our [Browser Quickstart](browser_getting_started.md)

## What You Need to Know

### VM Limitations

Devbox Cloud is free to use during the Open Beta period, subject to the following restrictions. These restrictions are applied to each GitHub User: 

#### Resources

* **CPU**: 1 Core, shared
* **RAM**: 2 GB
* **OS Image**: Alpine Linux 3.17.1 (amd64) 

#### Persistence + Concurrency

* **Max number of VMs**: 3 Concurrent VMs
* **VM Lifespan**: VMs will stay alive for up to 5 minutes after a user disconnects. After that point, the VM and all data will be deleted.

#### Localization

Devbox Cloud is available in the regions below. Devbox will attempt to provision your Cloud Shell in the data center closest to your current location. Note that using a VPN or other network obfuscator may cause you to connect to a different region. 

