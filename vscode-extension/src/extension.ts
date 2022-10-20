// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {

	// Use the console to output diagnostic information (console.log) and errors (console.error)
	// This line of code will only be executed once when your extension is activated
	vscode.window.onDidOpenTerminal(async (event) => {
		runDevboxShell();
	});
}

async function runDevboxShell() {

	const result = await vscode.workspace.findFiles('devbox.json');
	if (result.length > 0) {
		vscode.commands.executeCommand('workbench.action.terminal.sendSequence', {
			'text': 'devbox shell \r\n'
		});
	}
}

// This method is called when your extension is deactivated
export function deactivate() { }
