const NODE_MAJOR_VERSION = process.versions.node.split('.')[0];
if (!NODE_MAJOR_VERSION) {
  throw new Error('Node is not installed properly');
}
