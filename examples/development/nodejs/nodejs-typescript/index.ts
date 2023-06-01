const NODE_MAJOR_VERSION = process.versions.node.split('.')[0];
if (NODE_MAJOR_VERSION !== "18") {
  throw new Error('Node version is not 18');
} else {
  console.log('Node version is 18');
}
