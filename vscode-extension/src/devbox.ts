import { window, workspace, commands, ProgressLocation, Uri } from 'vscode';
import { promisify } from 'node:util';
import { ChildProcess, ForkOptions, SpawnOptions, execFile, fork, spawn, exec, spawnSync } from 'node:child_process';

// const exe = promisify(spawn);


export function devboxShellenv() {
    if (workspace.workspaceFolders) {
        let options: ForkOptions = {
            stdio: ['pipe', 'pipe', 'pipe', 'ipc'],
            cwd: "/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/"
        };
        const devboxProcess: ChildProcess = spawn('go', ['run', '/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/integrate.go',], options);
        devboxProcess.send({ hello: "child" });
        devboxProcess.on('close', (code: number) => {
            console.log('called close! with code: ' + code);
            console.log(devboxProcess.stdout);
            console.log(devboxProcess.stderr);

        });

        devboxProcess.on('message', (message: any, handle) => {
            console.log(message);
            if (message?.status === "finished") {
                devboxProcess.send('Closing');
                commands.executeCommand("workbench.action.closeWindow");
            }
        });
        console.log("ooooooooooooooo");
        // console.log(stdout);
        // console.log("+++++++++");
        // console.log(stderr);
    }
}

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
                        // await workspace.fs.stat(Uri.joinPath(workspaceUri, "devbox.json"));
                        const workingDir = workspace.workspaceFolders[0].uri;
                        const dotdevbox = Uri.joinPath(workingDir, '/.devbox');
                        const scriptsDir = Uri.joinPath(dotdevbox, '/gen/scripts');
                        const nodescript = Uri.joinPath(scriptsDir, 'vscode.js');
                        try {
                            // check if .devbox exists
                            const fsres = await workspace.fs.stat(dotdevbox);
                        } catch (error) {
                            //.devbox doesn't exist
                            // running devbox shellenv to create it
                            spawnSync('devbox', ['shellenv'], {
                                cwd: workingDir.path
                            });
                        }
                        // check if nodejs script exists
                        try {
                            const fsres = await workspace.fs.stat(nodescript);
                        } catch (error) {
                            //nodescript doesn't exist
                            // create nodescript file
                            const content = Buffer.from(nodeFileContent, 'utf8');
                            await workspace.fs.writeFile(nodescript, content);
                        }
                        // run script file
                        console.log(nodescript.path);

                        const nodeprocess: ChildProcess = spawn('node',
                            [nodescript.path],
                            {
                                env: {
                                    processenv: workingDir.path,
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
                            setTimeout(() => {
                                commands.executeCommand("workbench.action.closeWindow");
                            }, 1000);

                        });
                    }
                });
                return p;
            }
        );


        // setTimeout(() => {
        // }, 5000);


        //     // commands.executeCommand("workbench.action.closeWindow");
        //     console.log("2222222");
        //     console.log(stdout);
        //     console.log("+++++33333333++++");
        //     console.log(stderr);
        // }
    }
}

const nodeFileContent = `const child_process = require('child_process');

const devbox = '/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/devbox';
const cdir = '/Users/mohsenansari/code/jetpack/go.jetpack.io/examples/vscode/vscodetest/';
const code = 'code'
process.send({ status: process.env['processenv'] });
// allowing time for parent process to fully close
setTimeout(() => {
    let child = child_process.spawn(devbox, ['integrate', 'vscode'], {
        cwd: process.env['processenv'],
        stdio: [0, 1, 2, 'ipc']
    });


    child.on('close', (code) => {
        process.exit(code);
    });

    child.send({ hello: "child" });
    child.on('message', function (msg, handle) {
        if (msg.status === "finished") {
            console.log(msg);
        }
    });
}, 5000);
`;
