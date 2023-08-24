# Minikube + Argo Example

Run and test Argo Workflows on a local Minkube instance.

The init_hook in this example configures minikube to store it's data in a local `home/` directory, so your host kubeconfig is not affected by this shell. The scripts in this example do the following:

`minikube` starts minikube and tails it's logs
`install-argo` Installs Argo Workflows based on the Argo Quickstart documentation
`argo-port-forward` Forwards the port of the Argo deployment, so you can access the Argo UI at `https://localhost:2746` (note the `https`).

## Usage Instructions

Note: macOS users need to have Docker Desktop installed. This is because the Docker Daemon cannot run natively on macOS

1. Start `minikube` by running `devbox run minikube`. This will install and spin up minikube in a local shell, and then tail the logs
2. Install argo on minkube using `devbox run argo-install`
3. Forward the ports from your argo deployment using `devbox run argo-port-forward`
4. You can now run `devbox shell`, and use the Argo CLI to interact with Argo in minikube. You can also access the Argo UI at [https://localhost:2746](https://localhost:2746)
