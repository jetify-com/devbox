import { window, workspace, commands, ProgressLocation, Uri } from 'vscode';
import { spawn, spawnSync } from 'node:child_process';


interface Message {
    status: string
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

                const p = new Promise<void>(async (resolve, reject) => {

                    if (workspace.workspaceFolders) {
                        const workingDir = workspace.workspaceFolders[0].uri;
                        const dotdevbox = Uri.joinPath(workingDir, '/.devbox');
                        try {
                            // check if .devbox exists
                            await workspace.fs.stat(dotdevbox);
                        } catch (error) {
                            //.devbox doesn't exist
                            // running devbox shellenv to create it
                            spawnSync('devbox', ['shellenv'], {
                                cwd: workingDir.path
                            });
                        }
                        // run devbox integrate and then close this window
                        const devbox = '/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/devbox';
                        // const devbox = 'devbox'
                        let child = spawn(devbox, ['integrate', 'vscode'], {
                            cwd: workingDir.path,
                            stdio: [0, 1, 2, 'ipc']
                        });
                        child.on('close', (code: number) => {
                            if (code === 1) {
                                window.showErrorMessage("Failed to setup devbox environment.");
                                reject();
                            }
                        });
                        child.send({ configDir: workingDir.path });
                        child.on('message', function (msg: Message, handle) {
                            if (msg.status === "finished") {
                                resolve();
                                commands.executeCommand("workbench.action.closeWindow");
                            }
                        });

                    }
                });
                return p;
            }
        );
    }
}
