// The module 'vscode' contains the VS Code extensibility API
import { workspace, window, commands, Uri, ExtensionContext } from 'vscode';
import { posix } from 'path';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: ExtensionContext) {
	// This line of code will only be executed once when your extension is activated
	init();
	// Creating file watchers to watch for events on devbox.json
	const fswatcher = workspace.createFileSystemWatcher("**/devbox.json", false, false, false);
	fswatcher.onDidDelete(e => commands.executeCommand('setContext', 'devbox.configFileExists', false));
	fswatcher.onDidCreate(e => commands.executeCommand('setContext', 'devbox.configFileExists', true));
	fswatcher.onDidChange(e => checkDevboxJSON());

	// Check for devbox.json when a new folder is opened
	workspace.onDidChangeWorkspaceFolders(async (e) => checkDevboxJSON());

	// run devbox shell when terminal is opened
	window.onDidOpenTerminal(async (e) => {
		if (workspace.getConfiguration("devbox").get("autoShellOnTerminal") && e.name !== "DevboxTerminal") {
			await runInTerminal('devbox shell', true);
		}
	});

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

	const devboxRemove = commands.registerCommand('devboxNaNpxove', async () => {
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

	const setupDevcontainer = commands.registerCommand('devbox.setupDevContainer', async () => {
		await runInTerminal('devbox generate devcontainer', true);
	});
	const generateDockerfile = commands.registerCommand('devbox.generateDockerfile', async () => {
		await runInTerminal('devbox generate dockerfile', true);
	});

	context.subscriptions.push(devboxAdd);
	context.subscriptions.push(devboxRun);
	context.subscriptions.push(devboxInit);
	context.subscriptions.push(devboxRemove);
	context.subscriptions.push(devboxShell);
	context.subscriptions.push(setupDevcontainer);
	context.subscriptions.push(generateDockerfile);
}

async function init() {
	// check to activate a remote environment
	checkRemoteEnv();
	//check to activate devbox commands
	checkDevboxJSON();
}

async function checkRemoteEnv() {
	if (process.env["DEVBOX_OPEN_CLOUD_EDITOR"] === "1") {
		try {
			await commands.executeCommand("remote-tunnels.connectCurrentWindowToTunnel");
			// if (response === undefined) {
			// 	window.showErrorMessage("Couldn't connect to devbox cloud instance. Make sure to have 'Remote - Tunnels' extenstion installed and try again.");
			// 	await commands.executeCommand("workbench.extensions.search", "ms-vscode.remote-server");
			// }
		} catch (e) {
			window.showErrorMessage("Couldn't connect to devbox cloud instance. Make sure to have 'Remote - Tunnels' extenstion installed and try again.");
			await commands.executeCommand("workbench.extensions.installExtension", "ms-vscode.remote-server");
		}
	}
}


async function checkDevboxJSON() {
	// check if there is a workspace folder open
	if (workspace.workspaceFolders) {
		const workspaceUri = workspace.workspaceFolders[0].uri;
		try {
			// check if the folder has devbox.json in it
			await workspace.fs.stat(Uri.joinPath(workspaceUri, "devbox.json"));
			// devbox.json exists setcontext for devbox commands to be available
			commands.executeCommand('setContext', 'devbox.configFileExists', true);

		} catch (err) {
			console.log(err);
			// devbox.json does not exist
			commands.executeCommand('setContext', 'devbox.configFileExists', false);
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
