import * as path from 'path';
import { glob } from 'glob';

export async function run(): Promise<void> {
	// mocha 12+ is published as an ES module, so load it with a dynamic import
	// from this CommonJS test entrypoint.
	const mochaModule = await import('mocha');

	// Create the mocha test
	const mocha = new mochaModule.default({
		ui: 'tdd',
		color: true
	});

	const testsRoot = path.resolve(__dirname, '..');
	const files = await glob('**/**.test.js', { cwd: testsRoot });

	// Add files to the test suite
	files.forEach(f => mocha.addFile(path.resolve(testsRoot, f)));
	await mocha.loadFilesAsync();

	// Run the mocha test
	return new Promise((c, e) => {
		mocha.run(failures => {
			if (failures > 0) {
				e(new Error(`${failures} tests failed.`));
			} else {
				c();
			}
		});
	});
}
