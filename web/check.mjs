import { spawnSync } from 'node:child_process';
import { readdir } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
const root = fileURLToPath(new URL('.', import.meta.url));
for (const directory of ['', 'local/', 'remote/', 'help/', 'tools/']) {
  for (const name of await readdir(root + directory)) {
    if (!/\.(mjs|js)$/.test(name)) continue;
    const result = spawnSync(process.execPath, ['--check', root + directory + name], { encoding: 'utf8' });
    if (result.status) { process.stderr.write(result.stderr); process.exit(result.status); }
  }
}
console.log('All frontend JavaScript syntax checks passed. No bundler or install required.');
