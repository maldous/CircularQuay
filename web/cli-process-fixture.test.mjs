// Owned CI subprocesses only. Never copy a product module or rewrite its URL.
import { spawn } from 'node:child_process';
import { mkdtemp, writeFile, rm } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { pathToFileURL } from 'node:url';
import assert from 'node:assert/strict';

export function cliEnvironment() {
  const environment = { TZ: 'UTC' };
  for (const name of ['PATH', 'Path', 'SystemRoot', 'WINDIR', 'TEMP', 'TMP', 'TMPDIR', 'NODE_V8_COVERAGE', 'RUNNER_TRACKING_ID']) {
    if (process.env[name] !== undefined) environment[name] = process.env[name];
  }
  // Preserve an existing CI process-owner marker for runner teardown without
  // printing it or treating it as proof of cleanup. In particular, no NODE_OPTIONS, tokens, provider configuration or public
  // deployment origin can be inherited by these fixed local CLI contracts.
  return environment;
}

function supervisionFailure(code) {
  return Object.assign(new Error('Owned CLI child failed its bounded supervision contract.'), { code });
}

function killOwnedChild(child, posixGroup) {
  if (!Number.isInteger(child.pid) || child.pid <= 0 || child.pid === process.pid) return;
  try {
    if (posixGroup) process.kill(-child.pid, 'SIGKILL');
    else child.kill('SIGKILL');
  } catch (error) {
    if (error.code !== 'ESRCH') throw supervisionFailure('CQ_CLI_KILL_REFUSED');
  }
}

export function runCli(module, args = [], { cwd, preload, timeout = 60_000, descendants = false, outputLimit = 256 * 1024 } = {}) {
  assert.ok(Number.isInteger(timeout) && timeout > 0 && timeout <= 60_000);
  assert.ok(Number.isInteger(outputLimit) && outputLimit > 0 && outputLimit <= 256 * 1024);
  // The only descendant-producing contract is selected by the existing
  // Ubuntu landing/Sonar routes. Windows callers here are reviewed leaf CLIs;
  // direct Windows termination is NOT process-tree attestation.
  if (descendants && process.platform === 'win32') return Promise.reject(supervisionFailure('CQ_CLI_TREE_UNSUPPORTED'));
  const posixGroup = process.platform !== 'win32';
  const argv = preload ? ['--import', pathToFileURL(preload).href, module, ...args] : [module, ...args];
  const child = spawn(process.execPath, argv, { cwd, env: cliEnvironment(),
    stdio: ['ignore', 'pipe', 'pipe'], detached: posixGroup, windowsHide: true });
  return new Promise(resolve => {
    const parts = { stdout: [], stderr: [] }, sizes = { stdout: 0, stderr: 0 };
    let error, exitObserved = false, closed = false, complete = false, cleanupTimer, deadline;
    const result = (status, signal) => ({ status, signal, error, closed, reaped: exitObserved,
      treeStrategy: posixGroup ? 'owned-posix-group' : 'reviewed-windows-leaf-only',
      stdout: Buffer.concat(parts.stdout).toString('utf8'), stderr: Buffer.concat(parts.stderr).toString('utf8') });
    const finish = (status, signal) => {
      if (complete) return;
      complete = true;
      clearTimeout(deadline); clearTimeout(cleanupTimer);
      resolve(result(status, signal));
    };
    const stop = code => {
      if (complete || cleanupTimer) return;
      error ||= supervisionFailure(code);
      try { killOwnedChild(child, posixGroup); }
      catch (failure) { error = failure; }
      cleanupTimer = setTimeout(() => {
        // OS refusal is not success or proven cleanup. Bound the fixture wait,
        // close owned streams and report uncertainty; never signal other PIDs.
        error = supervisionFailure('CQ_CLI_CLEANUP_UNCERTAIN');
        child.stdout?.destroy(); child.stderr?.destroy(); child.unref();
        finish(child.exitCode, child.signalCode);
      }, 2_000);
    };
    const capture = (name, bytes) => {
      const room = outputLimit - sizes[name];
      if (room > 0) parts[name].push(bytes.subarray(0, room));
      sizes[name] += Math.min(room, bytes.length);
      if (bytes.length > room) stop('CQ_CLI_OUTPUT_LIMIT');
    };
    child.stdout?.on('data', bytes => capture('stdout', bytes));
    child.stderr?.on('data', bytes => capture('stderr', bytes));
    child.stdout?.on('error', () => stop('CQ_CLI_PIPE_ERROR'));
    child.stderr?.on('error', () => stop('CQ_CLI_PIPE_ERROR'));
    child.once('error', () => stop('CQ_CLI_SPAWN_ERROR'));
    child.once('exit', () => { exitObserved = true; });
    child.once('close', (status, signal) => { closed = true; finish(status, signal); });
    deadline = setTimeout(() => stop('CQ_CLI_TIMEOUT'), timeout);
  });
}


export function assertCliPassed(result, message) {
  assert.equal(result.error, undefined, message);
  assert.equal(result.closed, true, message);
  assert.equal(result.reaped, true, message);
  assert.equal(result.signal, null, message);
  assert.equal(result.status, 0, result.stderr || message);
  assert.equal(result.stderr, '', message);
}

export async function previewPreload(t, ports, signal = 'SIGTERM') {
  const root = await mkdtemp(join(tmpdir(), 'cq-default-preview-'));
  t.after(() => rm(root, { recursive: true, force: true }));
  const file = join(root, 'observe-preview.mjs');
  // This preload never changes argv, module identity, HTTP implementations,
  // ports or counters. It observes real fixed-loopback responses, then invokes
  // the child's real registered graceful callback. That is not OS-signal proof.
  const code = `import http from 'node:http';
const ports = ${JSON.stringify(ports)};
const signal = ${JSON.stringify(signal)};
const end = Date.now() + 15_000;
function request(port) {
  return new Promise((resolve, reject) => {
    const call = http.get({host:'127.0.0.1',port,path:'/',agent:false}, response => {
      let size=0; const chunks=[];
      response.on('data', part => { size+=part.length; if(size>262144) response.destroy(new Error('bounded response')); else chunks.push(part); });
      response.on('error', reject);
      response.on('end', () => {
        const html=Buffer.concat(chunks).toString('utf8');
        if(response.statusCode!==200 || !String(response.headers['content-type']).includes('text/html') || !html.includes('CircularQuay')) reject(new Error('unexpected synthetic preview'));
        else resolve();
      });
    });
    call.setTimeout(1000, () => call.destroy(new Error('bounded loopback timeout')));
    call.on('error', reject);
  });
}
async function observe() {
  try {
    for (const port of ports) await request(port);
    if (process.listenerCount(signal)===0) throw new Error('missing graceful callback');
    process.stdout.write('Owned default preview served all fixed ports; invoking graceful callback.\\n');
    process.emit(signal);
  } catch {
    if(Date.now()<end) setTimeout(observe,50);
    else { process.stderr.write('Owned default preview did not become ready.\\n'); process.exit(1); }
  }
}
setTimeout(observe,0);
`;
  await writeFile(file, code, { flag: 'wx', mode: 0o600 });
  return file;
}
