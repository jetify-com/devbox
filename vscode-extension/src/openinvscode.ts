import { Uri, commands, window } from 'vscode';
import fetch from 'node-fetch';
import { exec } from 'child_process';
import FormData = require('form-data');
import { open, writeFile } from 'fs/promises';

export async function handleOpenInVSCode(uri: Uri) {
    const queryParams = new URLSearchParams(uri.query);

    if (queryParams.has('vm_id') && queryParams.has('token')) {
        window.showInformationMessage('Setting up devbox');

        // getting ssh keys
        const response = await getVMInfo(queryParams.get('token'), queryParams.get('vm_id'));
        const res = await response.json();
        // TODO: remove debug
        console.log("data:\n");
        console.log(res);
        // set ssh config
        await setupSSHConfig(res['vm_id'], res['private_key']);
        // connect to remote vm
        connectToRemote(res['username'], res['vm_id']);
    } else {
        window.showErrorMessage('Error parsing information for remote environment.');
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
            // eslint-disable-next-line @typescript-eslint/naming-convention
            'Authorization': `Bearer ${token}`
        }
    });
    return response;
}

async function setupSSHConfig(vmId: string, prKey: string) {
    exec('./devbox generate ssh-config --config=./', (error, stdout, stderr) => {
        if (error) {
            console.error(`exec error: ${error}`);
            return;
        }
        console.log(`stdout: ${stdout}`);
        console.log(`stderr: ${stderr}`);
    });
    // save private key to file
    const prkeyPath = `${process.env['HOME']}/.config/devbox/ssh/keys/${vmId}.vm.devbox-vms.internal`;
    try {
        const prKeydata = new Uint8Array(Buffer.from(prKey));
        const fileHandler = await open(prkeyPath, 'w');
        await writeFile(fileHandler, prKeydata, { flag: 'w' });
        await fileHandler.close();
    } catch (err) {
        // When a request is aborted - err is an AbortError
        console.error(err);
    }
}

function connectToRemote(username: string, vmId: string) {
    const pathToFile = `/home/${username}/`;
    const host = `${vmId}.vm.devbox-vms.internal`;
    const workspaceURI = `vscode-remote://ssh-remote+${host}${pathToFile}`;
    const uriToOpen = Uri.parse(workspaceURI);
    window.showInformationMessage(uriToOpen.toString());
    commands.executeCommand("vscode.openFolder", uriToOpen, false);
}