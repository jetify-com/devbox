var NODE_MAJOR_VERSION = process.versions.node.split('.')[0];
if (NODE_MAJOR_VERSION !== "18") {
    throw new Error('Node version is not 18');
}
