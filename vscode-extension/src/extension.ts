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
		const devboxJson = await readDevboxJson(workspaceUri);
		await vscode.workspace.fs.createDirectory(vscode.Uri.joinPath(workspaceUri, '.devcontainer/'));
		// console.log(devboxJson);
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

// This method is called when your extension is deactivated
export function deactivate() { }
