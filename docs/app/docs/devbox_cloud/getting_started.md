# Getting Started from your Terminal

Devbox Cloud is a new way to create and run your Devbox Project in an isolated cloud environment. 

Use cases for Devbox Cloud include: 
* Testing out Packages or Scripts in an Isolated Linux Environment -- Preview different combinations or environments in a remote VM
* Easy Preview Environments for your project -- Contributors or developers can 
* Develop in a consistent environment from any Machine -- Log on to your Cloud Shell from anywhere, and develop in a consistent Dev environment anywhere in the world. Our VMs are deployed at the edge using Fly.io to provide a low-latency environment


:::note
Devbox Cloud is currently in Beta and under active development. 
::: 

## How It Works

### Prerequisites
Devbox Cloud Shell requires the following: 

* **Devbox 0.2.0 or higher.** If you do not have Nix installed on your machine, Devbox will install it with the default configuration for your OS 
* **A Github Account with an SSH Key Configured**. This is used by Devbox to authenticate and connect you to your Cloud VM.


### Step 1: Authenticate with Github

Devbox provides an easy password-less login flow using the SSH keys attached to your Github Account. If you do not have SSH keys configured with Github, follow the instructions here: [Connecting to Github with SSH](https://docs.github.com/en/enterprise-server@3.4/authentication/connecting-to-github-with-ssh/about-ssh)

When you run `devbox cloud shell`, Devbox will first attempt to infer your Github username from your local environment, and prompt you if a username cannot be found. 

Once Devbox has your username, it will authenticate you over SSH using the private/public keypair associated with your Github Account. 

:::note
All authentication is handled via SSH. Devbox never reads or stores your private key.
:::  

### Step 2: Launch your Devbox Shell in a Cloud VM

Once you are authenticated, Devbox will provision and start your Cloud Shell: 
1. First, we will provision a VM within your region and connect using SSH. 
2. Your local project files will be synced to the VM using Mutagen
3. Once your files are updated, Devbox will install your dependencies and start a `devbox shell` for your project

<!-- Diagram goes here -->

If you are using Devbox for the first time, this process may take over 1 minute to complete, depending on the size and number of your project's dependencies. Subsequent sessions will reuse your VM, and should boot up and start in a few seconds

#### Example

Let's create a simple project that uses Python 3.10 with Poetry to manage our packages. We'll start by running `devbox init` in our project directory, and then adding the packages:

```bash
devbox init 
```
```bash
devbox add python310 poetry
```

This should create a devbox.json in your directory that looks like the following: 

```json
{
  "packages": [
    "poetry",
    "python310"
  ],
  "shell": {
    "init_hook": null
  },
  "nixpkgs": {
    "commit": "52e3e80afff4b16ccb7c52e9f0f5220552f03d04"
  }
}
```
Now you can start your Cloud Shell by running `devbox cloud shell`

```md
Devbox Cloud
Remote development environments powered by Nix

✓ Created a virtual machine in Sunnyvale, California (US)
✓ File syncing started
✓ Connecting to virtual machine


Installing nix packages. This may take a while... done.
Starting a devbox shell...
...

(devbox) ~/src/devbox-cloud-test 💫 watching for changes
❯
```

You are now connected to your remote shell


### Step 3: Sync your Local Changes to Devbox Cloud

When you start your cloud session, your files are kept locally, and synchronized with your Devbox Cloud VM when changes are detected. This means you can use your favorite tools and editors to develop your project, while running in an isolated cloud environment. 

#### Example

Let's 

### Step 4: Test your Services with Port-forwarding