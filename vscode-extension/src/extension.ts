// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import { workspace, window, commands, extensions, Uri, ExtensionContext } from 'vscode';
import * as process from 'process';
import * as cp from 'child_process';
import * as util from 'util';
import { posix } from 'path';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: ExtensionContext) {
	// This line of code will only be executed once when your extension is activated
	initialCheckDevboxJSON();
	// Creating file watchers to watch for events on devbox.json
	const fswatcher = workspace.createFileSystemWatcher("**/devbox.json", false, false, false);
	fswatcher.onDidDelete(e => {
		commands.executeCommand('setContext', 'devbox.configFileExists', false);
	});
	fswatcher.onDidCreate(e => {
		commands.executeCommand('setContext', 'devbox.configFileExists', true);
	});
	fswatcher.onDidChange(e => {
		initialCheckDevboxJSON();
	});

	// Check for devbox.json when a new folder is opened
	workspace.onDidChangeWorkspaceFolders(async (e) => {
		initialCheckDevboxJSON();
	});

	// run devbox shell when terminal is opened
	window.onDidOpenTerminal(async (e) => {
		if (workspace.getConfiguration("devbox").get("autoShellOnTerminal")) {
			runDevboxShell();
		}
	});

	const setupDevcontainer = commands.registerCommand('devbox.setupDevContainer', async () => {
		const exec = util.promisify(cp.exec);
		// determining cpu architecture - needed for devcontainer dockerfile
		const { stdout, stderr } = await exec("uname -m");
		let cpuArch = stdout;
		if (stderr) {
			console.log(stderr);
			const response = await window.showErrorMessage(
				"Could not determine the CPU architecture type. Is your architecture type Apple M1/arm64?",
				"Yes",
				"No",
			);
			cpuArch = response === "Yes" ? "arm64" : "something else";
		}
		setupDevContainerFiles(cpuArch);

	});

	const devboxAdd = commands.registerCommand('devbox.add', async () => {
		const result = await window.showInputBox({
			value: '',
			placeHolder: 'Package to add to devbox. E.g., python39',
		});
		ensureTerminalExists();
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': `devbox add ${result}\r\n`
		});
	});

	const devboxShell = commands.registerCommand('devbox.shell', async () => {
		// todo: add support for --config path to devbox.json
		ensureTerminalExists();
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox shell\r\n'
		});
	});

	const devboxInstall = commands.registerCommand('devbox.install', async () => {
		// todo: add support for --config path to devbox.json
		ensureTerminalExists();
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox shell -- echo "installed packages!""\r\n'
		});
	});

	const devboxInit = commands.registerCommand('devbox.init', async () => {
		ensureTerminalExists();
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox init\r\n'
		});
		commands.executeCommand('setContext', 'devbox.configFileExists', true);
	});

	context.subscriptions.push(devboxAdd);
	context.subscriptions.push(devboxInit);
	context.subscriptions.push(devboxShell);
	context.subscriptions.push(devboxInstall);
	context.subscriptions.push(setupDevcontainer);
}

async function initialCheckDevboxJSON() {
	// check if there is a workspace folder open
	if (workspace.workspaceFolders) {
		const workspaceUri = workspace.workspaceFolders[0].uri;
		try {
			// check if the folder has devbox.json in it
			await workspace.fs.stat(Uri.joinPath(workspaceUri, "devbox.json"));
			// devbox.json exists setcontext for devbox commands to be available
			commands.executeCommand('setContext', 'devbox.configFileExists', true);

			if (workspace.getConfiguration("devbox").get("promptUpdateSettings")) {
				const response = await window.showInformationMessage(
					"A Devbox project is opened. Do you want to project settings with Devbox environment?",
					"Update Settings",
					"Don't show again"
				);
				if (response === "Update Settings") {
					const devboxJson = await readDevboxJson(workspaceUri);

					updateSettings(workspaceUri.path, devboxJson);
				} else if (response === "Don't show again") {
					workspace.getConfiguration("devbox").update("promptUpdateSettings", false);
				}
			}
		} catch (err) {
			console.log(err);
			// devbox.json does not exist
			commands.executeCommand('setContext', 'devbox.configFileExists', false);
			console.log("devbox.json does not exist");
		}
	}
}

function updateSettings(workspacePath: String, devboxJson: any) {
	// Updating process.env.PATH
	process.env["PATH"] = process.env["PATH"] + ":" + workspacePath + "/.devbox/nix/profile/default/bin";
	// Updating language extension settings
	// For now we only update Go, Python3, and Nodejs language extensions
	devboxJson["packages"].forEach((pkg: String) => {
		if (pkg.startsWith("python3")) {
			if (extensions.getExtension("ms-python.python")?.isActive) {
				workspace.getConfiguration("python").update("defaultInterpreterPath", workspacePath + "/.devbox/nix/profile/default/bin/python3");
			}
		}
		if (pkg.startsWith("go_1_") || pkg === "go") {
			if (extensions.getExtension("golang.go")?.isActive) {
				workspace.getConfiguration("go").update("gopath", workspacePath + "/.devbox/nix/profile/default/bin/go");
			}
		}
		if (pkg.startsWith("nodejs-") || pkg === "nodejs") {
			if (extensions.getExtension("eslint")?.isActive) {
				workspace.getConfiguration("eslint").update("nodepath", workspacePath + "/.devbox/nix/profile/default/bin/node");
			}
		}
		//TODO: add support for other common languages
	});

}

async function runDevboxShell() {
	const result = await workspace.findFiles('devbox.json');
	if (result.length > 0) {
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox shell\r\n'
		});

	}
}

async function setupDevContainerFiles(cpuArch: String) {
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
		console.error('there was an error', error);
	}
	// Display a message box to the user
}

async function readDevboxJson(workspaceUri: Uri) {
	const fileUri = workspaceUri.with({ path: posix.join(workspaceUri.path, 'devbox.json') });
	const readData = await workspace.fs.readFile(fileUri);
	const readStr = Buffer.from(readData).toString('utf8');
	const devboxJsonData = JSON.parse(readStr);
	return devboxJsonData;

}

function ensureTerminalExists(): void {
	// ensure a terminal is open
	// This check has to exist since there is no way for extension to run code in
	// the terminal, unless a terminal session is already open.
	if ((<any>window).terminals.length === 0) {
		window.showErrorMessage('No active terminals. Re-run the command without closing the opened terminal.');
		const terminal = window.createTerminal({ name: `Terminal` });
		terminal.show();
	}
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

// This method is called when your extension is deactivated
export function deactivate() { }
