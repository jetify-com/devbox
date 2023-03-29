import * as os from 'os';
import * as which from 'which';
import fetch from 'node-fetch';
import { exec } from 'child_process';
import * as FormData from 'form-data';
import { Uri, commands, window } from 'vscode';
import { chmod, open, writeFile } from 'fs/promises';

type VmInfo = {
    /* eslint-disable @typescript-eslint/naming-convention */
    vm_id: string;
    private_key: string;
    username: string;
    working_directory: string;
    /* eslint-enable @typescript-eslint/naming-convention */
};

export async function handleOpenInVSCode(uri: Uri) {
    const queryParams = new URLSearchParams(uri.query);

    if (queryParams.has('vm_id') && queryParams.has('token')) {
        //Not yet supported for windows + WSL - will be added in future
        if (os.platform() !== 'win32') {
            window.showInformationMessage('Setting up devbox');
            // getting ssh keys
            const response = await getVMInfo(queryParams.get('token'), queryParams.get('vm_id'));
            const res = await response.json() as VmInfo;
            console.debug("data:");
            console.debug(res);
            // set ssh config
            await setupSSHConfig(res.vm_id, res.private_key);
            // connect to remote vm
            connectToRemote(res.username, res.vm_id, res.working_directory);
        } else {
            window.showErrorMessage('This function is not yet supported in Windows operating system.');
        }
    } else {
        window.showErrorMessage('Error parsing information for remote environment.');
        console.debug(queryParams.toString());
    };
}

async function getVMInfo(token: string | null, vmId: string | null): Promise<any> {
    // send post request to gateway to get vm info and ssh keys
    const gatewayHost = 'https://api.devbox.sh/g/vm_info';
    const data = new FormData();
    data.append("vm_id", vmId);
    const response = await fetch(gatewayHost, {
        method: 'post',
        body: data,
        headers: {
            authorization: `Bearer ${token}`
        }
    });
    return response;
}

async function setupDevboxLauncher(): Promise<any> {
    // download devbox launcher script
    const gatewayHost = 'https://releases.jetpack.io/devbox';
    const response = await fetch(gatewayHost, {
        method: 'get',
    });
    const launcherPath = `${process.env['HOME']}/.config/devbox/launcher.sh`;

    try {
        const launcherScript = await response.text();
        const launcherData = new Uint8Array(Buffer.from(launcherScript));
        const fileHandler = await open(launcherPath, 'w');
        await writeFile(fileHandler, launcherData, { flag: 'w' });
        await chmod(launcherPath, 0o711);
        await fileHandler.close();
    } catch (err) {
        console.error(err);
    }
    return launcherPath;
}

async function setupSSHConfig(vmId: string, prKey: string) {

    const devboxBinary = await which('devbox', { nothrow: true });
    let devboxPath = 'devbox';
    if (devboxBinary === null) {
        devboxPath = await setupDevboxLauncher();
    }
    // For testing change devbox to path to a compiled devbox binary or add --config
    exec(`${devboxPath} generate ssh-config`, (error, stdout, stderr) => {
        if (error) {
            window.showErrorMessage('Failed to setup ssh config. Run VSCode in debug mode to see logs.');
            console.error(`Failed to setup ssh config: ${error}`);
            return;
        }
        console.debug(`stdout: ${stdout}`);
        console.debug(`stderr: ${stderr}`);
    });
    // save private key to file
    const prkeyPath = `${process.env['HOME']}/.config/devbox/ssh/keys/${vmId}.vm.devbox-vms.internal`;
    try {
        const prKeydata = new Uint8Array(Buffer.from(prKey));
        const fileHandler = await open(prkeyPath, 'w');
        await writeFile(fileHandler, prKeydata, { flag: 'w' });
        await chmod(prkeyPath, 0o600);
        await fileHandler.close();
    } catch (err) {
        // When a request is aborted - err is an AbortError
        window.showErrorMessage('Failed to setup ssh config. Run VSCode in debug mode to see logs.');
        console.error(err);
    }
}

function connectToRemote(username: string, vmId: string, workDir: string) {
    const host = `${username}@${vmId}.vm.devbox-vms.internal`;
    const workspaceURI = `vscode-remote://ssh-remote+${host}${workDir}`;
    const uriToOpen = Uri.parse(workspaceURI);
    console.debug("uriToOpen: ", uriToOpen.toString());
    commands.executeCommand("vscode.openFolder", uriToOpen, false);
}