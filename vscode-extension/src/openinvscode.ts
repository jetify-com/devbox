import { Uri, commands, window } from 'vscode';
import fetch from 'node-fetch';
import { exec } from 'child_process';
import * as FormData from 'form-data';
import { chmod, open, writeFile } from 'fs/promises';

export async function handleOpenInVSCode(uri: Uri) {
    const queryParams = new URLSearchParams(uri.query);

    if (queryParams.has('vm_id') && queryParams.has('token')) {
        window.showInformationMessage('Setting up devbox');

        // getting ssh keys
        type VmInfo = {
            /* eslint-disable @typescript-eslint/naming-convention */
            vm_id: string;
            private_key: string;
            username: string;
            /* eslint-enable @typescript-eslint/naming-convention */
        };
        const response = await getVMInfo(queryParams.get('token'), queryParams.get('vm_id'));
        const res = await response.json() as VmInfo;
        console.debug("data:");
        console.debug(res);
        // set ssh config
        await setupSSHConfig(res.vm_id, res.private_key);
        // connect to remote vm
        connectToRemote(res.username, res.vm_id);
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

async function setupSSHConfig(vmId: string, prKey: string) {
    // TODO: change this back before to devbox generate ssh-config
    // This requires a release for devbox that has generate ssh-config included in it
    // For testing change devbox to path to a compiled devbox binary or add --config
    exec('devbox generate ssh-config', (error, stdout, stderr) => {
        if (error) {
            console.error(`exec error: ${error}`);
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
        console.error(err);
    }
}

function connectToRemote(username: string, vmId: string) {
    const pathToFile = `/home/${username}/`;
    const host = `${username}@${vmId}.vm.devbox-vms.internal`;
    const workspaceURI = `vscode-remote://ssh-remote+${host}${pathToFile}`;
    const uriToOpen = Uri.parse(workspaceURI);
    console.debug("uriToOpen: ", uriToOpen.toString());
    commands.executeCommand("vscode.openFolder", uriToOpen, false);
}