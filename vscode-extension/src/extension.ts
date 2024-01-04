// The module 'vscode' contains the VS Code extensibility API
import { workspace, window, commands, Uri, ExtensionContext } from 'vscode';
import { posix } from 'path';

import { handleOpenInVSCode } from './openinvscode';
import { devboxReopen } from './devbox';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: ExtensionContext) {
	// This line of code will only be executed once when your extension is activated
	initialCheckDevboxJSON(context);
	// Creating file watchers to watch for events on devbox.json
	const fswatcher = workspace.createFileSystemWatcher("**/devbox.json", false, false, false);

	fswatcher.onDidDelete(e => {
		commands.executeCommand('setContext', 'devbox.configFileExists', false);
		context.workspaceState.update("configFileExists", false);
	});
	fswatcher.onDidCreate(e => {
		commands.executeCommand('setContext', 'devbox.configFileExists', true);
		context.workspaceState.update("configFileExists", true);
	});
	fswatcher.onDidChange(e => initialCheckDevboxJSON(context));

	// Check for devbox.json when a new folder is opened
	workspace.onDidChangeWorkspaceFolders(async (e) => initialCheckDevboxJSON(context));

	// run devbox shell when terminal is opened
	window.onDidOpenTerminal(async (e) => {
		if (workspace.getConfiguration("devbox").get("autoShellOnTerminal")
			&& e.name !== "DevboxTerminal"
			&& context.workspaceState.get("configFileExists")
		) {
			await runInTerminal('devbox shell', true);
		}
	});

	// open in vscode URI handler
	const handleVSCodeUri = window.registerUriHandler({ handleUri: handleOpenInVSCode });

	const devboxAdd = commands.registerCommand('devbox.add', async () => {
		const result = await window.showInputBox({
			value: '',
			placeHolder: 'Package to add to devbox. E.g., python39',
		});
		await runInTerminal(`devbox add ${result}`, false);
	});

	const devboxRun = commands.registerCommand('devbox.run', async () => {
		const items = await getDevboxScripts();
		if (items.length > 0) {
			const result = await window.showQuickPick(items);
			await runInTerminal(`devbox run ${result}`, true);
		} else {
			window.showInformationMessage("No scripts found in devbox.json");
		}
	});

	const devboxShell = commands.registerCommand('devbox.shell', async () => {
		// todo: add support for --config path to devbox.json
		await runInTerminal('devbox shell', true);
	});

	const devboxRemove = commands.registerCommand('devbox.remove', async () => {
		const items = await getDevboxPackages();
		if (items.length > 0) {
			const result = await window.showQuickPick(items);
			await runInTerminal(`devbox rm ${result}`, false);
		} else {
			window.showInformationMessage("No packages found in devbox.json");
		}
	});

	const devboxInit = commands.registerCommand('devbox.init', async () => {
		await runInTerminal('devbox init', false);
		commands.executeCommand('setContext', 'devbox.configFileExists', true);
	});

	const devboxInstall = commands.registerCommand('devbox.install', async () => {
		await runInTerminal('devbox install', true);
	});

	const devboxUpdate = commands.registerCommand('devbox.update', async () => {
		await runInTerminal('devbox update', true);
	});

	const devboxSearch = commands.registerCommand('devbox.search', async () => {
		const result = await window.showInputBox({ placeHolder: "Name or a subset of a name of a package to search" });
		await runInTerminal(`devbox search ${result}`, true);
	});

	const setupDevcontainer = commands.registerCommand('devbox.setupDevContainer', async () => {
		await runInTerminal('devbox generate devcontainer', true);
	});

	const generateDockerfile = commands.registerCommand('devbox.generateDockerfile', async () => {
		await runInTerminal('devbox generate dockerfile', true);
	});

	const reopen = commands.registerCommand('devbox.reopen', async () => {
		await devboxReopen();
	});

	context.subscriptions.push(reopen);
	context.subscriptions.push(devboxAdd);
	context.subscriptions.push(devboxRun);
	context.subscriptions.push(devboxInit);
	context.subscriptions.push(devboxInstall);
	context.subscriptions.push(devboxSearch);
	context.subscriptions.push(devboxUpdate);
	context.subscriptions.push(devboxRemove);
	context.subscriptions.push(devboxShell);
	context.subscriptions.push(setupDevcontainer);
	context.subscriptions.push(generateDockerfile);
	context.subscriptions.push(handleVSCodeUri);
}

async function initialCheckDevboxJSON(context: ExtensionContext) {
	// check if there is a workspace folder open
	if (workspace.workspaceFolders) {
		const workspaceUri = workspace.workspaceFolders[0].uri;
		try {
			// check if the folder has devbox.json in it
			await workspace.fs.stat(Uri.joinPath(workspaceUri, "devbox.json"));
			// devbox.json exists setcontext for devbox commands to be available
			commands.executeCommand('setContext', 'devbox.configFileExists', true);
			context.workspaceState.update("configFileExists", true);
		} catch (err) {
			console.log(err);
			// devbox.json does not exist
			commands.executeCommand('setContext', 'devbox.configFileExists', false);
			context.workspaceState.update("configFileExists", false);
			console.log("devbox.json does not exist");
		}
	}
}

async function runInTerminal(cmd: string, showTerminal: boolean) {
	// check if a terminal is open
	if ((<any>window).terminals.length === 0) {
		const terminalName = 'DevboxTerminal';
		const terminal = window.createTerminal({ name: terminalName });
		if (showTerminal) {
			terminal.show();
		}
		terminal.sendText(cmd, true);
	} else {
		// A terminal is open
		// run the given cmd in terminal
		await commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': `${cmd}\r\n`
		});
	}
}

async function getDevboxScripts(): Promise<string[]> {
	try {
		if (!workspace.workspaceFolders) {
			window.showInformationMessage('No folder or workspace opened');
			return [];
		}
		const workspaceUri = workspace.workspaceFolders[0].uri;
		const devboxJson = await readDevboxJson(workspaceUri);
		return Object.keys(devboxJson['shell']['scripts']);
	} catch (error) {
		console.error('Error processing devbox.json - ', error);
		return [];
	}
}

async function getDevboxPackages(): Promise<string[]> {
	try {
		if (!workspace.workspaceFolders) {
			window.showInformationMessage('No folder or workspace opened');
			return [];
		}
		const workspaceUri = workspace.workspaceFolders[0].uri;
		const devboxJson = await readDevboxJson(workspaceUri);
		return devboxJson['packages'];
	} catch (error) {
		console.error('Error processing devbox.json - ', error);
		return [];
	}
}

async function readDevboxJson(workspaceUri: Uri) {
	const fileUri = workspaceUri.with({ path: posix.join(workspaceUri.path, 'devbox.json') });
	const readData = await workspace.fs.readFile(fileUri);
	const readStr = Buffer.from(readData).toString('utf8');
	const devboxJsonData = JSON.parse(readStr);
	return devboxJsonData;
}

// This method is called when your extension is deactivated
export function deactivate() { }
