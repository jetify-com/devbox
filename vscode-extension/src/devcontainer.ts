import { workspace, window, Uri } from 'vscode';
import { posix } from 'path';

export async function setupDevContainerFiles(cpuArch: String) {
    try {
        if (!workspace.workspaceFolders) {
            return window.showInformationMessage('No folder or workspace opened');
        }
        const workspaceUri = workspace.workspaceFolders[0].uri;
        const devcontainerUri = Uri.joinPath(workspaceUri, '.devcontainer/');
        // Parsing devbox.json data
        const devboxJson = await readDevboxJson(workspaceUri);
        // creating .devcontainer directory and its files
        await workspace.fs.createDirectory(devcontainerUri);
        const dockerfileContent = getDockerfileContent();
        await workspace.fs.writeFile(
            Uri.joinPath(devcontainerUri, 'Dockerfile'),
            Buffer.from(dockerfileContent, 'utf8')
        );

        const devContainerJSON = getDevcontainerJSON(devboxJson, cpuArch);
        await workspace.fs.writeFile(
            Uri.joinPath(devcontainerUri, 'devcontainer.json'),
            Buffer.from(devContainerJSON, 'utf8')
        );
    } catch (error) {
        console.error('Error processing devbox.json - ', error);
        window.showErrorMessage('Error processing devbox.json');
    }
}

export async function readDevboxJson(workspaceUri: Uri) {
    const fileUri = workspaceUri.with({ path: posix.join(workspaceUri.path, 'devbox.json') });
    const readData = await workspace.fs.readFile(fileUri);
    const readStr = Buffer.from(readData).toString('utf8');
    const devboxJsonData = JSON.parse(readStr);
    return devboxJsonData;

}



function getDockerfileContent(): String {
    return `
	# See here for image contents: https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/debian/.devcontainer/base.Dockerfile

	# [Choice] Debian version (use bullseye on local arm64/Apple Silicon): bullseye, buster
	ARG VARIANT="buster"
	FROM mcr.microsoft.com/vscode/devcontainers/base:0-\${VARIANT}

	# These dependencies are required by Nix.
	RUN apt update -y
	RUN apt -y install --no-install-recommends curl xz-utils

	USER vscode

	# Install nix
	ARG NIX_INSTALL_SCRIPT=https://nixos.org/nix/install
	RUN curl -fsSL \${NIX_INSTALL_SCRIPT} | sh -s -- --no-daemon
	ENV PATH /home/vscode/.nix-profile/bin:\${PATH}

	# Install devbox
	RUN sudo mkdir /devbox && sudo chown vscode /devbox
	RUN curl -fsSL https://get.jetpack.io/devbox | bash -s -- -f

	# Setup devbox environment
	COPY --chown=vscode ./devbox.json /devbox/devbox.json
	RUN devbox shell --config /devbox/devbox.json -- echo "Nix Store Populated"
	ENV PATH /devbox/.devbox/nix/profile/default/bin:\${PATH}
	ENTRYPOINT devbox shell
	`;
}

function getDevcontainerJSON(devboxJson: any, cpuArch: String): String {

    let devcontainerObject: any = {};
    devcontainerObject = {
        // For format details, see https://aka.ms/devcontainer.json. For config options, see the README at:
        // https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/debian
        "name": "Devbox Remote Container",
        "build": {
            "dockerfile": "./Dockerfile",
            // Update 'VARIANT' to pick a Debian version: bullseye, buster
            // Use bullseye on local arm64/Apple Silicon.
            "args": {
                "VARIANT": cpuArch.trim() === "arm64" ? "bullseye" : "buster"
            }
        },
        "customizations": {
            "vscode": {
                "settings": {
                    // Add custom vscode settings for remote environment here
                },
                "extensions": [
                    // Add custom vscode extensions for remote environment here
                ]
            }
        },
        // Comment out to connect as root instead. More info: https://aka.ms/vscode-remote/containers/non-root.
        "remoteUser": "vscode"
    };

    devboxJson["packages"].forEach((pkg: String) => {
        if (pkg.includes("python3")) {
            devcontainerObject.customizations.vscode.settings["python.defaultInterpreterPath"] = "/devbox/.devbox/nix/profile/default/bin/python3";
            devcontainerObject.customizations.vscode.extensions.push("ms-python.python");
        }
        if (pkg.includes("go_1_") || pkg === "go") {
            devcontainerObject.customizations.vscode.extensions.push("golang.go");
        }
        //TODO: add support for other common languages
    });

    return JSON.stringify(devcontainerObject, null, 4);
}
