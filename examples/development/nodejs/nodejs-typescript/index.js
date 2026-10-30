var NODE_MAJOR_VERSION = process.versions.node.split('.')[0];
if (NODE_MAJOR_VERSION !== "24") {
    throw new Error('Node version is not 24');
}
else {
    console.log('Node version is 24');
}
