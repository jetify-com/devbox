# Minikube + Helm + Kubectl Example

Run Helm + Kubernetes locally using Minikube in a Devbox shell.

The init_hook in this example configures minikube + helm to store their data in a local `home/` directory, so your host kubeconfig and helm repos are not affected by this shell.

## Usage Instructions

Note: macOS users need to have Docker Desktop installed. This is because the Docker Daemon cannot run natively on macOS

1. Start `minikube` by running `devbox run minikube`. This will install and spin up minikube in a local shell, and then tail the logs
2. In a different terminal, create a new shell with `devbox shell`.
3. You can now deploy to minikube using `kubectl` or `helm`.
4. To shutdown minikube, use CTRL-C to terminate the shell where you started minikube.
