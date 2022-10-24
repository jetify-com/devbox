// The module 'vscode' contains the VS Code extensibility API
// Import the module and reference it with the alias vscode in your code below
import * as vscode from 'vscode';
import * as fs from 'fs';
import { config } from 'process';

type ScriptList = string[];
type ConfigScripts = Map<string,ScriptList>;

interface ShellConfig {
	initHook: string[]
	scripts: Map<string,Map<string, string | string[]>>
}

interface DevboxJson {
	packages: string[]
	shell: ShellConfig
}

var configScripts: ConfigScripts = new Map();

// This method is called when your extension is activated
// Your extension is activated the very first time the command is executed
export function activate(context: vscode.ExtensionContext) {
	vscode.workspace.findFiles('devbox.json').then(
		(uris: vscode.Uri[]) => {
			uris.forEach(uri => {
				vscode.workspace.openTextDocument(uri).then((document) => {
					let text = document.getText();
					let data: DevboxJson = JSON.parse(text);
					console.log(data);
					configScripts.set("devbox.json", [...Object.keys(data.shell.scripts)]);
				  });
				
			});
		}
	);

	context.subscriptions.push(vscode.commands.registerCommand("devbox.runScript", runScript));

	
	// Use the console to output diagnostic information (console.log) and errors (console.error)
	// This line of code will only be executed once when your extension is activated
	// vscode.window.onDidOpenTerminal(async (event) => {
	// 	runDevboxShell();
	// });
}

function runScript(){
	var option: vscode.QuickPickItem;
	const quickPick = vscode.window.createQuickPick();

	quickPick.items = configScripts.get('devbox.json').map(
		label => ({label})
	);

	quickPick.onDidAccept(() => {
		let terminal = vscode.window.createTerminal();
		terminal.show();
		terminal.sendText(`~/devbox/devbox run ${option.label} \r\n`);
	});

	quickPick.onDidChangeSelection(selection => {
		if (selection[0]){
			option = selection[0];
		}
	});

	quickPick.onDidHide(() => quickPick.dispose);

	quickPick.show();



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
