import { spawnSync } from 'node:child_process';
import { readdir, realpath } from 'node:fs/promises';
import { fileURLToPath } from 'node:url';
import { join } from 'node:path';

export async function checkFrontend(root) {
  for (const directory of ['', 'local/', 'remote/', 'help/', 'tools/']) {
    for (const name of await readdir(join(root, directory))) {
      if (!/\.(mjs|js)$/.test(name)) continue;
      const result = spawnSync(process.execPath, ['--check', join(root, directory, name)], { encoding: 'utf8' });
      if (result.error) throw result.error;
      if (result.status !== 0) return { status: result.status ?? 1, stderr: result.stderr || '' };
    }
  }
  return { status: 0, stderr: '' };
}

export async function isCheckEntry(entry = process.argv[1]) {
  if (!entry) return false;
  try { return await realpath(entry) === await realpath(fileURLToPath(import.meta.url)); }
  catch (error) { if (error.code === 'ENOENT') return false; throw error; }
}
export async function runFrontendCheck(root, {
  error = text => process.stderr.write(text),
  line = text => console.log(text),
  setExit = status => { process.exitCode = status; },
} = {}) {
  const result = await checkFrontend(root);
  if (result.status) { error(result.stderr); setExit(result.status); }
  else line('All frontend JavaScript syntax checks passed. No bundler or install required.');
  return result.status;
}
if (await isCheckEntry()) await runFrontendCheck(fileURLToPath(new URL('.', import.meta.url)));
