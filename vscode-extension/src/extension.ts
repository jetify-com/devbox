// The module 'vscode' contains the VS Code extensibility API
import * as util from 'util';
import * as cp from 'child_process';
import { workspace, window, commands, Uri, ExtensionContext, QuickPickItem } from 'vscode';
import { setupDevContainerFiles, readDevboxJson } from './devcontainer';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: ExtensionContext) {
	// This line of code will only be executed once when your extension is activated
	initialCheckDevboxJSON();
	// Creating file watchers to watch for events on devbox.json
	const fswatcher = workspace.createFileSystemWatcher("**/devbox.json", false, false, false);
	fswatcher.onDidDelete(e => commands.executeCommand('setContext', 'devbox.configFileExists', false));
	fswatcher.onDidCreate(e => commands.executeCommand('setContext', 'devbox.configFileExists', true));
	fswatcher.onDidChange(e => initialCheckDevboxJSON());

	// Check for devbox.json when a new folder is opened
	workspace.onDidChangeWorkspaceFolders(async (e) => initialCheckDevboxJSON());

	// run devbox shell when terminal is opened
	window.onDidOpenTerminal(async (e) => {
		if (workspace.getConfiguration("devbox").get("autoShellOnTerminal")) {
			await runInTerminal('devbox shell');
		}
	});

	const devboxAdd = commands.registerCommand('devbox.add', async () => {
		const result = await window.showInputBox({
			value: '',
			placeHolder: 'Package to add to devbox. E.g., python39',
		});
		await runInTerminal(`devbox add ${result}`);
	});

	const devboxRun = commands.registerCommand('devbox.run', async () => {
		const items = await getDevboxScripts();
		if (items.length > 0) {
			const result = await window.showQuickPick(items);
			await runInTerminal(`devbox run ${result}`);
		} else {
			window.showInformationMessage("No scripts found in devbox.json");
		}
	});

	const devboxShell = commands.registerCommand('devbox.shell', async () => {
		// todo: add support for --config path to devbox.json
		await runInTerminal('devbox shell');
	});

	const devboxRemove = commands.registerCommand('devbox.remove', async () => {
		const items = await getDevboxPackages();
		if (items.length > 0) {
			const result = await window.showQuickPick(items);
			await runInTerminal(`devbox rm ${result}`);
		} else {
			window.showInformationMessage("No packages found in devbox.json");
		}
	});

	const devboxInit = commands.registerCommand('devbox.init', async () => {
		await runInTerminal('devbox init');
		commands.executeCommand('setContext', 'devbox.configFileExists', true);
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
			cpuArch = response === "Yes" ? "arm64" : "undefined";
		}
		await setupDevContainerFiles(cpuArch);

	});

	context.subscriptions.push(devboxAdd);
	context.subscriptions.push(devboxRun);
	context.subscriptions.push(devboxInit);
	context.subscriptions.push(devboxShell);
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

		} catch (err) {
			console.log(err);
			// devbox.json does not exist
			commands.executeCommand('setContext', 'devbox.configFileExists', false);
			console.log("devbox.json does not exist");
		}
	}
}

async function runInTerminal(cmd: string) {
	// check if a terminal is open
	if ((<any>window).terminals.length === 0) {
		const terminal = window.createTerminal({ name: `Terminal` });
		terminal.show();
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

// This method is called when your extension is deactivated
export function deactivate() { }
