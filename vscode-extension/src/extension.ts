// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import * as cp from 'child_process';
import * as util from 'util';
import { posix } from 'path';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {

	// Use the console to output diagnostic information (console.log) and errors (console.error)
	// This line of code will only be executed once when your extension is activated
	vscode.window.onDidOpenTerminal(async (event) => {
		runDevboxShell();
	});

	const setupDevcontainer = vscode.commands.registerCommand('devbox.setupDevContainer', async () => {
		return setupDevContainer();
	});

	context.subscriptions.push(setupDevcontainer);
}

async function runDevboxShell() {
	const exec = util.promisify(cp.exec);
	const result = await vscode.workspace.findFiles('devbox.json');
	if (result.length > 0) {
		// const { stdout, stderr } = await exec('devbox shell');
		// console.log('stdout:', stdout);
		// console.log('stderr:', stderr);
		let response = "test";
		response = await vscode.commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox shell\r\nopen .\r\nexit\r\n'
		});
		setTimeout(() => { vscode.commands.executeCommand('workbench.action.closeWindow'); }, 10000);

	}
}

async function setupDevContainer() {
	try {
		if (!vscode.workspace.workspaceFolders) {
			return vscode.window.showInformationMessage('No folder or workspace opened');
		}
		const workspaceUri = vscode.workspace.workspaceFolders[0].uri;
		const devcontainerUri = vscode.Uri.joinPath(workspaceUri, '.devcontainer/');
		// Parsing devbox.json data
		const devboxJson = await readDevboxJson(workspaceUri);
		// creating .devcontainer directory and its files
		await vscode.workspace.fs.createDirectory(devcontainerUri);
		const dockerfileContent = getDockerfileContent();
		await vscode.workspace.fs.writeFile(
			vscode.Uri.joinPath(devcontainerUri, 'Dockerfile'),
			Buffer.from(dockerfileContent, 'utf8')
		);

		const devContainerJSON = getDevcontainerJSON();
		await vscode.workspace.fs.writeFile(
			vscode.Uri.joinPath(devcontainerUri, 'devcontainer.json'),
			Buffer.from(devContainerJSON, 'utf8')
		);

		vscode.window.showInformationMessage(devboxJson["packages"].toString());
	} catch (error) {
		console.error('there was an error', error);
	}
	// Display a message box to the user
}

async function readDevboxJson(workspaceUri: vscode.Uri) {
	const fileUri = workspaceUri.with({ path: posix.join(workspaceUri.path, 'devbox.json') });
	const readData = await vscode.workspace.fs.readFile(fileUri);
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
	
	# install devbox
	RUN sudo mkdir /devbox && sudo chown vscode /devbox
	RUN curl -fsSL https://get.jetpack.io/devbox | FORCE=1 bash
	COPY --chown=vscode devbox.json /devbox/devbox.json
	RUN devbox shell --config /devbox/devbox.json -- echo "Nix Store Populated"
	ENV PATH /devbox/.devbox/nix/profile/default/bin:\${PATH}
	`;
}

function getDevcontainerJSON(): String {
	return `
	// For format details, see https://aka.ms/devcontainer.json. For config options, see the README at:
	// https://github.com/microsoft/vscode-dev-containers/tree/v0.245.2/containers/debian
	{
		"name": "Debian",
		"build": {
			"dockerfile": "./Dockerfile",
			// Update 'VARIANT' to pick an Debian version: bullseye, buster
			// Use bullseye on local arm64/Apple Silicon.
			"args": {
				"VARIANT": "bullseye"
			}
		},
		"customizations": {
			"vscode": {
				"settings": {
					"python.defaultInterpreterPath": "/devbox/.devbox/nix/profile/default/bin/python3"
				},
				"extensions": [
					"ms-python.python",
					"golang.go"
				]
			}
		},
		// Comment out to connect as root instead. More info: https://aka.ms/vscode-remote/containers/non-root.
		"remoteUser": "vscode"
	}`;
}

// This method is called when your extension is deactivated
export function deactivate() { }
