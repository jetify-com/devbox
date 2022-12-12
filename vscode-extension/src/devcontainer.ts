import { workspace, window, Uri } from 'vscode';
import { posix } from 'path';

export async function setupDevContainerFiles() {
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

        const devContainerJSON = getDevcontainerJSON(devboxJson);
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
    FROM alpine:3

    # Setting up devbox user
    ENV DEVBOX_USER=devbox
    RUN adduser -h /home/$DEVBOX_USER -D -s /bin/bash $DEVBOX_USER
    RUN addgroup sudo
    RUN addgroup $DEVBOX_USER sudo
    RUN echo " $DEVBOX_USER      ALL=(ALL:ALL) NOPASSWD: ALL" >> /etc/sudoers
    
    # installing dependencies
    RUN apk add --no-cache bash binutils git libstdc++ xz sudo
    
    USER $DEVBOX_USER
    
    # installing devbox
    RUN wget --quiet --output-document=/dev/stdout https://get.jetpack.io/devbox | bash -s -- -f
    RUN chown -R "\${DEVBOX_USER}:\${DEVBOX_USER}" /usr/local/bin/devbox
    
    # nix installer script
    RUN wget --quiet --output-document=/dev/stdout https://nixos.org/nix/install | sh -s -- --no-daemon
    RUN . ~/.nix-profile/etc/profile.d/nix.sh
    # updating PATH
    ENV PATH="/home/\${DEVBOX_USER}/.nix-profile/bin:/home/\${DEVBOX_USER}/.devbox/nix/profile/default/bin:\${PATH}"
    
    WORKDIR /code
    COPY devbox.json devbox.json
    RUN devbox shell -- echo "Installing packages"
    ENTRYPOINT ["devbox"]
    CMD ['shell']    
	`;
}

function getDevcontainerJSON(devboxJson: any): String {

    let devcontainerObject: any = {};
    devcontainerObject = {
        // For format details, see https://aka.ms/devcontainer.json. For config options, see the README at:
        // https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/debian
        "name": "Devbox Remote Container",
        "build": {
            "dockerfile": "./Dockerfile",
            "context": "..",
        },
        "customizations": {
            "vscode": {
                "settings": {
                    // Add custom vscode settings for remote environment here
                },
                "extensions": [
                    "jetpack-io.devbox"
                    // Add custom vscode extensions for remote environment here
                ]
            }
        },
        // Comment out to connect as root instead. More info: https://aka.ms/vscode-remote/containers/non-root.
        "remoteUser": "devbox"
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
