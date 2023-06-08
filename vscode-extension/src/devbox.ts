import { window, workspace, commands, ProgressLocation, Uri } from 'vscode';
import { ChildProcess, spawn, spawnSync } from 'node:child_process';

// const exe = promisify(spawn);

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
                        const scriptsDir = Uri.joinPath(dotdevbox, '/gen/scripts');
                        const nodescript = Uri.joinPath(scriptsDir, 'vscode.js');
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
                        // check if nodejs script exists
                        try {
                            await workspace.fs.stat(nodescript);
                        } catch (error) {
                            //nodescript doesn't exist
                            // create nodescript file
                            const content = Buffer.from(nodeFileContent, 'utf8');
                            await workspace.fs.writeFile(nodescript, content);
                        }
                        // run script file
                        const nodeprocess: ChildProcess = spawn('node',
                            [nodescript.path],
                            {
                                env: {
                                    devboxDir: workingDir.path,
                                    ...process.env
                                },
                                cwd: workingDir.path,
                                stdio: [0, 1, 2, 'ipc']
                            }
                        );
                        nodeprocess.on('close', (code: number) => {
                            console.log('called close! with code: ' + code);
                        });

                        nodeprocess.on('message', (message: Message, handler) => {
                            console.log(message.status);
                            resolve();
                            commands.executeCommand("workbench.action.closeWindow");

                        });
                    }
                });
                return p;
            }
        );
    }
}

const nodeFileContent = `const child_process = require('child_process');

const devbox = '/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/devbox';
process.send({ status: process.env['devboxDir'] });
// allowing time for parent process to fully close
let child = child_process.spawn(devbox, ['integrate', 'vscode'], {
    cwd: process.env['devboxDir'],
    stdio: [0, 1, 2, 'ipc']
});
child.on('close', (code) => {
    process.exit(code);
});
child.send({ configDir: process.env['devboxDir'] });
child.on('message', function (msg, handle) {
    if (msg.status === "finished") {
        console.log(msg);
    }
});

`;
