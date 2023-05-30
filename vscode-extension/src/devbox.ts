import { window, workspace, commands, ProgressLocation } from 'vscode';
import { promisify } from 'node:util';
import { ForkOptions, SpawnOptions, execFile, fork, spawn } from 'node:child_process';


const exe = promisify(execFile);


export async function devboxShellenv() {
    if (workspace.workspaceFolders) {
        const options: ForkOptions = {
            stdio: ['pipe', 'pipe', 'pipe', 'ipc']
        };
        const { stdout, stderr } = await exe('code', ['/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/.devbox'], options);
        commands.executeCommand("workbench.action.closeWindow");
        console.log("ooooooooooooooo");
        console.log(stdout);
        console.log("+++++++++");
        console.log(stderr);
    }
}

export async function devboxReopen() {
    if (workspace.workspaceFolders) {
        await window.withProgress({
            location: ProgressLocation.Notification,
            title: "Setting up your Devbox environment. Please don't close vscode.",
            cancellable: true
        },
            async (progress, token) => {
                token.onCancellationRequested(() => {
                    console.log("User canceled the long running operation");
                });

                const p = new Promise<void>(resolve => {
                    setTimeout(() => {
                        resolve();
                    }, 5000);
                });
                return p;
            }
        );

        const { stdout, stderr } = await exe('code', ['/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/'], { env: { moo: 'COWMOO', ...process.env } });
        setTimeout(() => {
            commands.executeCommand("workbench.action.closeWindow");
        }, 5000);


        // commands.executeCommand("workbench.action.closeWindow");
        console.log("2222222");
        console.log(stdout);
        console.log("+++++33333333++++");
        console.log(stderr);
    }
}